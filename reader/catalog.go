package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveRef(ref pdfcpu.IndirectRef) pdfcpu.Object {
	// 7.3.10
	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	entry, found := r.xref.FindTableEntryForIndRef(&ref)
	if !found || entry.Free {
		return nil
	}
	return entry.Object
}

func (r resolver) resolveFormField(o pdfcpu.Object) (*model.FormField, error) {
	var err error
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		// did we already resolve this value ?
		if ff := r.formFields[ref]; ff != nil {
			return ff, nil
		}
		// we haven't resolved it yet: do it
		o = r.resolveRef(ref)
	}
	if o == nil {
		return nil, nil
	}
	f, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return nil, fmt.Errorf("unexpected type for FormField: %T", o)
	}
	var fi model.FormField
	if typ := f.NameEntry("FT"); typ != nil {
		fi.Ft = model.FormType(*typ)
	}
	if t := f.StringEntry("T"); t != nil {
		fi.T = *t
	}
	if as := f.NameEntry("AS"); as != nil {
		fi.AS = model.Name(*as)
	}
	if ff := f.IntEntry("Ff"); ff != nil {
		fi.Ff = model.FormFlag(*ff)
	}
	if ml := f.IntEntry("MaxLen"); ml != nil {
		fi.MaxLen = *ml
	} else {
		fi.MaxLen = -1
	}
	if f := f.IntEntry("F"); f != nil {
		fi.F = *f
	}
	if da := f.StringLiteralEntry("DA"); da != nil {
		fi.DA = decodeStringLit(*da)
	}

	fi.Rect = rectangleFromArray(f.ArrayEntry("Rect"))

	contents, _ := f["Contents"].(pdfcpu.StringLiteral)
	fi.Contents = decodeStringLit(contents)

	fi.AP, err = r.resolveAppearanceDict(f["AP"])
	if err != nil {
		return nil, err
	}

	if isRef { // write back to the cache
		r.formFields[ref] = &fi
	}
	return &fi, nil
}

func rectangleFromArray(ar pdfcpu.Array) model.Rectangle {
	if len(ar) < 4 {
		return model.Rectangle{}
	}
	llx, _ := ar[0].(pdfcpu.Float)
	lly, _ := ar[1].(pdfcpu.Float)
	urx, _ := ar[2].(pdfcpu.Float)
	ury, _ := ar[3].(pdfcpu.Float)
	return model.Rectangle{Llx: llx.Value(), Lly: lly.Value(), Urx: urx.Value(), Ury: ury.Value()}
}

func matrixFromArray(ar pdfcpu.Array) *model.Matrix {
	if len(ar) < 6 {
		return nil
	}
	var out model.Matrix
	for i := range out {
		f, _ := ar[i].(pdfcpu.Float)
		out[i] = float64(f)
	}
	return &out
}

func (r resolver) resolveAppearanceDict(o pdfcpu.Object) (*model.AppearanceDict, error) {
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		if ff := r.appearanceDicts[ref]; ff != nil {
			return ff, nil
		}
		o = r.resolveRef(ref)
	}
	if o == nil {
		return nil, nil
	}
	a, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return nil, fmt.Errorf("unexpected type for AppearanceDict: %T", o)
	}
	var (
		out model.AppearanceDict
		err error
	)
	if ap := a["N"]; ap != nil {
		out.N, err = r.resolveAppearanceEntry(ap)
		if err != nil {
			return nil, err
		}
	}
	if ap := a["R"]; ap != nil {
		out.R, err = r.resolveAppearanceEntry(ap)
		if err != nil {
			return nil, err
		}
	}
	if ap := a["D"]; ap != nil {
		out.D, err = r.resolveAppearanceEntry(ap)
		if err != nil {
			return nil, err
		}
	}
	if isRef { // write back to the cache
		r.appearanceDicts[ref] = &out
	}
	return &out, nil
}

func (r resolver) resolveAppearanceEntry(obj pdfcpu.Object) (*model.AppearanceEntry, error) {
	refApp, isRef := obj.(pdfcpu.IndirectRef)
	if isRef {
		obj = r.resolveRef(refApp)
	}
	out := make(model.AppearanceEntry)
	var err error
	// obj might be either a subdictionary or a streamdictionary
	switch obj := obj.(type) {
	case pdfcpu.Dict: // subdictionary
		if ap := r.appearanceEntries[refApp]; isRef && ap != nil {
			return ap, nil
		}
		for name, stream := range obj {
			refStream, isStreamRef := stream.(pdfcpu.IndirectRef)
			ap := r.xObjects[refStream]
			if isStreamRef && ap != nil {
				// nothing to do
			} else {
				if isStreamRef {
					stream = r.resolveRef(refStream)
				}
				streamDict, ok := stream.(pdfcpu.StreamDict)
				if !ok {
					return nil, fmt.Errorf("unexpected type for Stream object: %T", stream)
				}
				ap, err = r.processXObject(streamDict)
				if err != nil {
					return nil, err
				}
				if isStreamRef {
					r.xObjects[refStream] = ap
				}
			}
			out[model.Name(name)] = ap
		}
		r.appearanceEntries[refApp] = &out
	case pdfcpu.StreamDict: // stream
		ap := r.xObjects[refApp]
		if isRef && ap != nil {
			// nothing to do
		} else {
			ap, err = r.processXObject(obj)
			if err != nil {
				return nil, err
			}
			if isRef {
				r.xObjects[refApp] = ap
			}
		}
		out = model.AppearanceEntry{"": ap}
	default:
		return nil, fmt.Errorf("unexpected type for Appearance: %T", obj)
	}
	return &out, nil
}

func (r resolver) processXObject(obj pdfcpu.StreamDict) (*model.XObject, error) {
	var ap model.XObject
	ap.BBox = rectangleFromArray(obj.Dict.ArrayEntry("BBox"))
	ap.Matrix = matrixFromArray(obj.Dict.ArrayEntry("Matrix"))
	if res := obj.Dict["Resources"]; res != nil {
		var err error
		ap.Resources, err = r.resolveResources(res)
		if err != nil {
			return nil, err
		}
	}
	if err := obj.Decode(); err != nil {
		return nil, fmt.Errorf("can't decode Xobject stream: %w", err)
	}
	ap.Content = obj.Content
	return &ap, nil
}

func catalog(xref *pdfcpu.XRefTable) (model.Catalog, error) {
	var out model.Catalog
	d, err := xref.Catalog()
	if err != nil {
		return out, fmt.Errorf("can't resolve Catalog: %w", err)
	}
	r := resolver{
		xref:              xref,
		appearanceDicts:   make(map[pdfcpu.IndirectRef]*model.AppearanceDict),
		appearanceEntries: make(map[pdfcpu.IndirectRef]*model.AppearanceEntry),
		formFields:        make(map[pdfcpu.IndirectRef]*model.FormField),
		resources:         make(map[pdfcpu.IndirectRef]*model.ResourcesDict),
		xObjects:          make(map[pdfcpu.IndirectRef]*model.XObject),
	}

	out.AcroForm, err = r.processAcroForm(d)
	if err != nil {
		return out, err
	}
	return out, nil
}

func (r resolver) processAcroForm(catalog pdfcpu.Dict) (model.AcroForm, error) {
	var out model.AcroForm
	if ref := catalog["AcroForm"]; ref != nil {
		form, err := r.xref.DereferenceDict(ref)
		if err != nil {
			return out, fmt.Errorf("can't resolve Catalog.AcroForm: %w", err)
		}
		fields := form.ArrayEntry("Fields")
		out.Fields = make([]*model.FormField, len(fields))
		for i, f := range fields {
			ff, err := r.resolveFormField(f)
			if err != nil {
				return out, err
			}
			out.Fields[i] = ff
		}
		if na := form.BooleanEntry("NeedAppearances"); na != nil {
			out.NeedAppearances = *na
		}
	}
	return out, nil
}
