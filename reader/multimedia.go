package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveRendition(obj pdfcpu.Object) (out model.RenditionDict, err error) {
	objDict, _ := r.resolve(obj).(pdfcpu.Dict)
	n, _ := isString(r.resolve(objDict["N"]))
	out.N = decodeTextString(n)

	mh, _ := r.resolve(objDict["MH"]).(pdfcpu.Dict)
	out.MH, err = r.resolveMediaCriteria(mh["C"])
	if err != nil {
		return out, err
	}

	be, _ := r.resolve(objDict["BE"]).(pdfcpu.Dict)
	out.BE, err = r.resolveMediaCriteria(be["C"])
	if err != nil {
		return out, err
	}

	switch name := r.resolve(objDict["S"]); name {
	case pdfcpu.Name("MR"):
		out.Subtype, err = r.resolveRenditionMedia(objDict)
	case pdfcpu.Name("SR"):
		out.Subtype, err = r.resolveRenditionSelector(objDict)
	default:
		return out, errType("Rendition.S", name)
	}
	return out, err
}

func (r resolver) resolveMediaCriteria(obj pdfcpu.Object) (*model.MediaCriteria, error) {
	objDict, _ := r.resolve(obj).(pdfcpu.Dict)
	if objDict == nil {
		return nil, nil
	}
	var out model.MediaCriteria
	if b, ok := r.resolveBool(objDict["A"]); ok {
		out.A = model.ObjBool(b)
	}
	if b, ok := r.resolveBool(objDict["C"]); ok {
		out.C = model.ObjBool(b)
	}
	if b, ok := r.resolveBool(objDict["O"]); ok {
		out.O = model.ObjBool(b)
	}
	if b, ok := r.resolveBool(objDict["S"]); ok {
		out.S = model.ObjBool(b)
	}
	if i, ok := r.resolveInt(objDict["R"]); ok {
		out.R = model.ObjInt(i)
	}

	dDict, _ := r.resolve(objDict["D"]).(pdfcpu.Dict)
	out.D.V, _ = r.resolveInt(dDict["V"])
	out.D.M, _ = r.resolveInt(dDict["M"])

	zDict, _ := r.resolve(objDict["Z"]).(pdfcpu.Dict)
	v, _ := r.resolveArray(zDict["V"])
	if len(v) == 2 {
		out.Z.V[0], _ = r.resolveInt(v[0])
		out.Z.V[1], _ = r.resolveInt(v[1])
	}
	out.Z.M, _ = r.resolveInt(zDict["M"])

	var err error
	sids, _ := r.resolveArray(objDict["V"])
	out.V = make([]model.SoftwareIdentifier, len(sids))
	for i, sid := range sids {
		out.V[i], err = r.resolveSoftwareIdentifier(sid)
		if err != nil {
			return nil, err
		}
	}

	ps, _ := r.resolveArray(objDict["P"])
	if L := len(ps); L >= 1 {
		out.P[0], _ = r.resolveName(ps[0])
		if L >= 2 {
			out.P[1], _ = r.resolveName(ps[1])
		}
	}

	lArr, _ := r.resolveArray(objDict["L"])
	out.L = make([]string, len(lArr))
	for i, o := range lArr {
		ls, _ := isString(r.resolve(o))
		out.L[i] = decodeTextString(ls)
	}

	return &out, nil
}

func (r resolver) resolveSoftwareIdentifier(sid pdfcpu.Object) (out model.SoftwareIdentifier, err error) {
	sid = r.resolve(sid)
	sidDict, ok := sid.(pdfcpu.Dict)
	if !ok {
		return out, errType("MediaCriteria.V element", sid)
	}
	out.U, _ = isString(r.resolve(sidDict["U"]))

	l, _ := r.resolveArray(sidDict["L"])
	out.L = make([]int, len(l))
	for i, in := range l {
		out.L[i], _ = r.resolveInt(in)
	}

	h, _ := r.resolveArray(sidDict["H"])
	out.H = make([]int, len(h))
	for i, in := range h {
		out.H[i], _ = r.resolveInt(in)
	}

	if b, ok := r.resolveBool(sidDict["LI"]); ok {
		out.LI = model.ObjBool(b)
	}
	if b, ok := r.resolveBool(sidDict["HI"]); ok {
		out.HI = model.ObjBool(b)
	}

	os, _ := r.resolveArray(sidDict["OS"])
	out.OS = make([]string, len(os))
	for i, o := range os {
		out.OS[i], _ = isString(r.resolve(o))
	}
	return out, nil
}

func (r resolver) resolveRenditionMedia(d pdfcpu.Dict) (out model.RenditionMedia, err error) {
	out.C, err = r.resolveMediaClipDict(d["C"])
	if err != nil {
		return out, err
	}
	pDict, _ := r.resolve(d["P"]).(pdfcpu.Dict)
	out.P.BE = r.resolveMediaPlayerParameters(pDict["BE"])
	out.P.MH = r.resolveMediaPlayerParameters(pDict["MH"])

	if spDict, ok := r.resolve(d["SP"]).(pdfcpu.Dict); ok {
		be := r.resolveMediaScreenParameters(spDict["BE"])
		out.SP.BE = &be
		mh := r.resolveMediaScreenParameters(spDict["MH"])
		out.SP.MH = &mh
	}
	return out, nil
}

func (r resolver) resolveMediaClipDict(obj pdfcpu.Object) (out model.MediaClipDict, err error) {
	c, _ := r.resolve(obj).(pdfcpu.Dict)

	n, _ := isString(r.resolve(c["N"]))
	out.N = decodeTextString(n)

	switch kind := r.resolve(c["S"]); kind {
	case pdfcpu.Name("MCD"):
		out.Subtype, err = r.resolveRenditionMediaClipData(c)
	case pdfcpu.Name("MCS"):
		out.Subtype, err = r.resolveRenditionMediaClipSection(c)
	default:
		err = errType("RenditionMedia.S", kind)
	}
	return out, err
}

func (r resolver) resolveRenditionMediaClipSection(d pdfcpu.Dict) (out model.MediaClipSection, err error) {
	out.D, err = r.resolveMediaClipDict(d["D"])
	if err != nil {
		return out, err
	}
	out.Alt, err = r.resolveLanguageText(d["Alt"])
	if err != nil {
		return out, err
	}
	out.MH, err = r.resolveMediaClipSectionLimit(d["MH"])
	if err != nil {
		return out, err
	}
	out.BE, err = r.resolveMediaClipSectionLimit(d["BE"])
	if err != nil {
		return out, err
	}
	return out, nil
}

func (r resolver) resolveMediaClipSectionLimit(obj pdfcpu.Object) (out model.MediaClipSectionLimits, err error) {
	objDict, _ := r.resolve(obj).(pdfcpu.Dict)

	out.B, err = r.resolveMediaOffset(objDict["B"])
	if err != nil {
		return out, err
	}
	out.E, err = r.resolveMediaOffset(objDict["E"])
	if err != nil {
		return out, err
	}
	return out, nil
}

func (r resolver) resolveMediaOffset(offset pdfcpu.Object) (model.MediaOffset, error) {
	b, _ := r.resolve(offset).(pdfcpu.Dict)
	if b == nil {
		return nil, nil
	}
	switch name := r.resolve(b["S"]); name {
	case pdfcpu.Name("T"):
		tDict, _ := r.resolve(b["T"]).(pdfcpu.Dict)
		fl, _ := r.resolveNumber(tDict["V"])
		return model.ObjFloat(fl), nil
	case pdfcpu.Name("F"):
		i, _ := r.resolveInt(b["F"])
		return model.ObjInt(i), nil
	case pdfcpu.Name("M"):
		m, _ := isString(r.resolve(b["M"]))
		return model.ObjStringLiteral(decodeTextString(m)), nil
	default:
		return nil, errType("MediaOffset", name)
	}
}

func (r resolver) resolveRenditionMediaClipData(d pdfcpu.Dict) (out model.MediaClipData, err error) {
	// we first resolve to find the type
	dDict, _ := r.resolve(d["D"]).(pdfcpu.Dict)
	if subtype, _ := r.resolveName(dDict["Subtype"]); subtype == "Form" { // form Xobject
		out.D, err = r.resolveOneXObjectForm(d["D"]) // pass the indirect ref
	} else { // assume file spec
		out.D, err = r.resolveFileSpec(d["D"])
	}
	if err != nil {
		return out, err
	}

	out.CT, _ = isString(r.resolve(d["CT"]))

	p, _ := r.resolve(d["P"]).(pdfcpu.Dict)
	ps, _ := isString(r.resolve(p["TF"]))
	switch ps := model.MediaClipPermission(ps); ps {
	case "", model.TempAccess, model.TempAlways, model.TempExtract, model.TempNever:
		out.P = ps
	default:
		return out, errType("MediaClipData.P", p)
	}

	out.Alt, err = r.resolveLanguageText(d["Alt"])
	if err != nil {
		return out, err
	}

	bh, _ := r.resolve(d["BE"]).(pdfcpu.Dict)
	out.BE, _ = isString(r.resolve(bh["BU"]))

	mh, _ := r.resolve(d["MH"]).(pdfcpu.Dict)
	out.MH, _ = isString(r.resolve(mh["BU"]))

	return out, nil
}

func (r resolver) resolveLanguageText(obj pdfcpu.Object) (model.LanguageArray, error) {
	objAr, _ := r.resolveArray(obj)
	if len(objAr)%2 != 0 {
		return nil, fmt.Errorf("odd length for multi-language text array")
	}
	out := make(model.LanguageArray, len(objAr)/2)
	for i := range out {
		s1, _ := isString(r.resolve(objAr[2*i]))
		s2, _ := isString(r.resolve(objAr[2*i+1]))
		out[i] = [2]string{decodeTextString(s1), decodeTextString(s2)}
	}
	return out, nil
}

func (r resolver) resolveRenditionSelector(dict pdfcpu.Dict) (out model.RenditionSelector, err error) {
	arr, _ := r.resolveArray(dict["R"])
	out.R = make([]model.RenditionDict, len(arr))
	for i, re := range arr {
		out.R[i], err = r.resolveRendition(re)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

func (r resolver) resolveMediaPlayerParameters(obj pdfcpu.Object) (out model.MediaPlayerParameters) {
	objDict, _ := r.resolve(obj).(pdfcpu.Dict)
	if v, ok := r.resolveInt(objDict["V"]); ok {
		out.V = model.ObjInt(v)
	}
	out.C, _ = r.resolveBool(objDict["C"])
	if f, ok := r.resolveInt(objDict["F"]); ok {
		out.F = model.ObjInt(f)
	}
	dDict, _ := r.resolve(objDict["D"]).(pdfcpu.Dict)
	switch s, _ := r.resolveName(dDict["S"]); s {
	case "I":
		out.D = model.MediaDurationIntrinsic
	case "F":
		out.D = model.MediaDurationInfinite
	case "T":
		tDict, _ := r.resolve(dDict["T"]).(pdfcpu.Dict)
		if t, ok := r.resolveNumber(tDict["V"]); ok {
			out.D = model.ObjFloat(t)
		}
	}
	if a, ok := r.resolveBool(objDict["A"]); ok {
		out.NotAuto = !a
	}
	if t, ok := r.resolveNumber(objDict["RC"]); ok {
		out.RC = model.ObjFloat(t)
	}
	return out
}

func (r resolver) resolveMediaScreenParameters(obj pdfcpu.Object) (out model.MediaScreenParams) {
	objDict, _ := r.resolve(obj).(pdfcpu.Dict)
	if w, ok := r.resolveInt(objDict["W"]); ok {
		out.W = model.ObjInt(w)
	}
	b, _ := r.resolveArray(objDict["B"])
	if len(b) == 3 {
		fls := r.processFloatArray(b)
		var b [3]Fl
		copy(b[:], fls)
		out.B = &b
	}
	if f, ok := r.resolveNumber(objDict["O"]); ok {
		out.O = model.ObjFloat(f)
	}
	out.M, _ = r.resolveInt(objDict["M"])

	if fDict, ok := r.resolve(objDict["F"]).(pdfcpu.Dict); ok {
		out.F = new(model.MediaScreenFloatingWindow)

		d, _ := r.resolveArray(fDict["D"])
		if len(d) == 2 {
			out.F.D[0], _ = r.resolveInt(d[0])
			out.F.D[1], _ = r.resolveInt(d[1])
		}

		rt, _ := r.resolveInt(fDict["RT"])
		out.F.RT = uint8(rt)

		rt, _ = r.resolveInt(fDict["R"])
		out.F.R = uint8(rt)

		if w, ok := r.resolveInt(fDict["P"]); ok {
			out.F.P = model.ObjInt(w)
		}
		if w, ok := r.resolveInt(fDict["O"]); ok {
			out.F.O = model.ObjInt(w)
		}

		if w, ok := r.resolveBool(fDict["T"]); ok {
			out.F.T = model.ObjBool(w)
		}
		if w, ok := r.resolveBool(fDict["UC"]); ok {
			out.F.UC = model.ObjBool(w)
		}
		out.F.TT, _ = r.resolveLanguageText(fDict["TT"])
	}
	return out
}
