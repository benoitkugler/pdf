package reader

import (
	"errors"
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveXObjects(obj pdfcpu.Object) (map[model.ObjName]model.XObject, error) {
	obj = r.resolve(obj)
	if obj == nil {
		return nil, nil
	}
	objDict, isDict := obj.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("XObjects Dict", obj)
	}
	objMap := make(map[model.ObjName]model.XObject)
	for name, xObject := range objDict {
		xObjectModel, err := r.resolveOneXObject(xObject)
		if err != nil {
			return nil, err
		}
		if xObjectModel == nil { // ignore the name
			continue
		}
		objMap[model.ObjName(name)] = xObjectModel
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

// TODO: add test
// returns an error if img is nil
func (r resolver) resolveOneXObjectImage(img pdfcpu.Object) (*model.XObjectImage, error) {
	imgRef, isRef := img.(pdfcpu.IndirectRef)
	if imgModel := r.images[imgRef]; isRef && imgModel != nil {
		return imgModel, nil
	}
	var (
		out    model.XObjectImage
		stream pdfcpu.StreamDict
		err    error
	)
	out.Image, stream, err = r.resolveImage(img)
	if err != nil {
		return nil, err
	}

	out.ColorSpace, err = r.resolveOneColorSpace(stream.Dict["ColorSpace"])
	if err != nil {
		return nil, err
	}

	if maskRef, isRef := stream.Dict["Mask"]; isRef { // image mask
		out.Mask, err = r.resolveOneXObjectImage(maskRef)
		if err != nil {
			return nil, err
		}
	} else { // nil or colour mask
		mask, _ := r.resolveArray(stream.Dict["Mask"])
		if len(mask)%2 != 0 {
			return nil, fmt.Errorf("expected even length for array, got %v", mask)
		}
		outMask := make(model.MaskColor, len(mask)/2)
		for i := range outMask {
			a, _ := r.resolveInt(mask[2*i])
			b, _ := r.resolveInt(mask[2*i+1])
			outMask[i] = [2]int{a, b}
		}
		out.Mask = outMask
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
		out.SMask, err = r.resolveImageSMask(smask)
		if err != nil {
			return nil, err
		}
	}
	if s, ok := r.resolveInt(stream.Dict["SMaskInData"]); ok {
		out.SMaskInData = uint8(s)
	}
	if st, ok := r.resolveInt(stream.Dict["StructParent"]); ok {
		out.StructParent = model.ObjInt(st)
	}

	if isRef {
		r.images[imgRef] = &out
	}
	return &out, nil
}

func (r resolver) resolveImage(img pdfcpu.Object) (out model.Image, stream pdfcpu.StreamDict, err error) {
	img = r.resolve(img)
	stream, isStream := img.(pdfcpu.StreamDict)
	if !isStream {
		return out, stream, errType("Image", img)
	}
	cs, err := r.resolveStream(stream)
	if err != nil {
		return out, stream, err
	}
	if cs == nil {
		return out, stream, errors.New("missing stream for image")
	}
	out.Stream = *cs

	if w, ok := r.resolveInt(stream.Dict["Width"]); ok {
		out.Width = w
	}
	if h, ok := r.resolveInt(stream.Dict["Height"]); ok {
		out.Height = h
	}
	if b, ok := r.resolveInt(stream.Dict["BitsPerComponent"]); ok {
		out.BitsPerComponent = uint8(b)
	}
	if intent, ok := r.resolveName(stream.Dict["Intent"]); ok {
		out.Intent = model.ObjName(intent)
	}
	if m, ok := r.resolveBool(stream.Dict["ImageMask"]); ok {
		out.ImageMask = m
	}
	decode, _ := r.resolveArray(stream.Dict["Decode"])
	if !out.ImageMask {
		out.Decode, err = r.processPoints(decode)
		if err != nil {
			return out, stream, err
		}
	} else { // special case: [0 1] or [1 0]
		if len(decode) == 2 {
			var ra [2]Fl
			ra[0], _ = r.resolveNumber(decode[0])
			ra[1], _ = r.resolveNumber(decode[1])
			out.Decode = [][2]Fl{ra}
		}
		// else: ignore nil or invalid
	}
	if i, ok := r.resolveBool(stream.Dict["Interpolate"]); ok {
		out.Interpolate = bool(i)
	}
	return out, stream, nil
}

func (r resolver) resolveImageSMask(img pdfcpu.Object) (*model.ImageSMask, error) {
	imgRef, isRef := img.(pdfcpu.IndirectRef)
	if imgModel := r.imageSMasks[imgRef]; isRef && imgModel != nil {
		return imgModel, nil
	}
	var (
		out    model.ImageSMask
		stream pdfcpu.StreamDict
		err    error
	)
	out.Image, stream, err = r.resolveImage(img)
	if err != nil {
		return nil, err
	}

	matte, _ := r.resolveArray(stream.Dict["Matte"])
	out.Matte = r.processFloatArray(matte)

	if isRef {
		r.imageSMasks[imgRef] = &out
	}
	return &out, nil
}
