package reader

import (
	"errors"
	"fmt"
	"log"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveXObjects(obj pdfcpu.Object) (map[model.Name]model.XObject, error) {
	obj = r.resolve(obj)
	if obj == nil {
		return nil, nil
	}
	objDict, isDict := obj.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("XObjects Dict", obj)
	}
	objMap := make(map[model.Name]model.XObject)
	for name, xObject := range objDict {
		xObjectModel, err := r.resolveOneXObject(xObject)
		if err != nil {
			return nil, err
		}
		if xObjectModel == nil { // ignore the name
			continue
		}
		objMap[model.Name(name)] = xObjectModel
	}
	return objMap, nil
}

func (r resolver) resolveOneXObject(obj pdfcpu.Object) (model.XObject, error) {
	// we have to resolve the object first to find it's type
	// then it will be resolved once again in each sub function
	// we keep track of the reference
	objRes := r.resolve(obj)
	stream, ok := objRes.(pdfcpu.StreamDict)
	if !ok {
		return nil, errType("XObject", obj)
	}
	name, _ := stream.Dict["Subtype"].(pdfcpu.Name)
	switch name {
	case "Image":
		return r.resolveOneXObjectImage(obj)
	case "Form":
		return r.resolveOneXObjectForm(obj)
	default:
		return nil, fmt.Errorf("invalid XObject subtype %s", name)
	}
}

// returns an error if img is nil
func (r resolver) resolveOneXObjectImage(img pdfcpu.Object) (*model.XObjectImage, error) {
	imgRef, isRef := img.(pdfcpu.IndirectRef)
	if imgModel := r.images[imgRef]; isRef && imgModel != nil {
		return imgModel, nil
	}
	img = r.resolve(img)
	stream, isStream := img.(pdfcpu.StreamDict)
	if !isStream {
		log.Println(img)
		return nil, errType("Image", img)
	}
	cs, err := r.processContentStream(stream)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, errors.New("missing stream for image")
	}
	out := model.XObjectImage{ContentStream: *cs}

	if w := stream.Dict.IntEntry("Width"); w != nil {
		out.Width = *w
	}
	if h := stream.Dict.IntEntry("Height"); h != nil {
		out.Height = *h
	}
	out.ColorSpace, err = r.resolveOneColorSpace(stream.Dict["ColorSpace"])
	if err != nil {
		return nil, err
	}
	if b := stream.Dict.IntEntry("BitsPerComponent"); b != nil {
		out.BitsPerComponent = uint8(*b)
	}
	if intent := stream.Dict.NameEntry("Intent"); intent != nil {
		out.Intent = model.Name(*intent)
	}
	if m := stream.Dict.BooleanEntry("ImageMask"); m != nil {
		out.ImageMask = *m
	}
	// TODO:
	if mask := stream.Dict["Mask"]; mask != nil {
		fmt.Println("TODO: Mask", mask)
	}
	decode := stream.Dict.ArrayEntry("Decode")
	if !out.ImageMask {
		out.Decode, err = processRange(decode)
		if err != nil {
			return nil, err
		}
	} else { // special case: [0 1] or [1 0]
		if len(decode) == 2 {
			var r model.Range
			r[0], _ = isNumber(decode[0])
			r[1], _ = isNumber(decode[1])
			out.Decode = []model.Range{r}
		}
		// else: ignore nil or invalid
	}
	if i := stream.Dict.BooleanEntry("Interpolate"); i != nil {
		out.Interpolate = *i
	}
	alts := stream.Dict.ArrayEntry("Alternates")
	out.Alternates = make([]model.AlternateImage, len(alts))
	for i, alt := range alts {
		alt = r.resolve(alt) // the AlternateImage is itself cheap, don't other tracking its ref
		altDict, isDict := alt.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("AlternateImage", alt)
		}
		out.Alternates[i].Image, err = r.resolveOneXObjectImage(altDict["Image"])
		if err != nil {
			return nil, err
		}
		if b := altDict.BooleanEntry("DefaultForPrinting"); b != nil {
			out.Alternates[i].DefaultForPrinting = *b
		}
	}
	if smask := stream.Dict["SMask"]; smask != nil {
		out.SMask, err = r.resolveOneXObjectImage(smask)
		if err != nil {
			return nil, err
		}
	}
	if s := stream.Dict.IntEntry("SMaskInData"); s != nil {
		out.SMaskInData = uint8(*s)
	}

	if isRef {
		r.images[imgRef] = &out
	}
	return &out, nil
}
