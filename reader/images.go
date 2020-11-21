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
	name, _ := r.resolveName(stream.Dict["Subtype"])
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
	cs, err := r.resolveStream(stream)
	if err != nil {
		return nil, err
	}
	if cs == nil {
		return nil, errors.New("missing stream for image")
	}
	out := model.XObjectImage{Stream: *cs}

	if w, ok := r.resolveInt(stream.Dict["Width"]); ok {
		out.Width = w
	}
	if h, ok := r.resolveInt(stream.Dict["Height"]); ok {
		out.Height = h
	}
	out.ColorSpace, err = r.resolveOneColorSpace(stream.Dict["ColorSpace"])
	if err != nil {
		return nil, err
	}
	if b, ok := r.resolveInt(stream.Dict["BitsPerComponent"]); ok {
		out.BitsPerComponent = uint8(b)
	}
	if intent, ok := r.resolveName(stream.Dict["Intent"]); ok {
		out.Intent = model.Name(intent)
	}
	if m, ok := r.resolveBool(stream.Dict["ImageMask"]); ok {
		out.ImageMask = m
	}
	// TODO:
	if mask := r.resolve(stream.Dict["Mask"]); mask != nil {
		fmt.Println("TODO: Mask", mask)
	}
	decode, _ := r.resolveArray(stream.Dict["Decode"])
	if !out.ImageMask {
		out.Decode, err = r.processPoints(decode)
		if err != nil {
			return nil, err
		}
	} else { // special case: [0 1] or [1 0]
		if len(decode) == 2 {
			var ra [2]float64
			ra[0], _ = r.resolveNumber(decode[0])
			ra[1], _ = r.resolveNumber(decode[1])
			out.Decode = [][2]float64{ra}
		}
		// else: ignore nil or invalid
	}
	if i, ok := r.resolveBool(stream.Dict["Interpolate"]); ok {
		out.Interpolate = bool(i)
	}
	alts, _ := r.resolveArray(stream.Dict["Alternates"])
	out.Alternates = make([]model.AlternateImage, len(alts))
	for i, alt := range alts {
		alt = r.resolve(alt) // the AlternateImage is itself cheap, don't bother tracking its ref
		altDict, isDict := alt.(pdfcpu.Dict)
		if !isDict {
			return nil, errType("AlternateImage", alt)
		}
		out.Alternates[i].Image, err = r.resolveOneXObjectImage(altDict["Image"])
		if err != nil {
			return nil, err
		}
		if b, ok := r.resolveBool(altDict["DefaultForPrinting"]); ok {
			out.Alternates[i].DefaultForPrinting = b
		}
	}
	if smask := stream.Dict["SMask"]; smask != nil {
		out.SMask, err = r.resolveOneXObjectImage(smask)
		if err != nil {
			return nil, err
		}
	}
	if s, ok := r.resolveInt(stream.Dict["SMaskInData"]); ok {
		out.SMaskInData = uint8(s)
	}
	if st, ok := r.resolveInt(stream.Dict["StructParent"]); ok {
		out.StructParent = model.Int(st)
	}

	if isRef {
		r.images[imgRef] = &out
	}
	return &out, nil
}
