package reader

import (
	"fmt"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) resolveFormField(o pdfcpu.Object) (*model.FormField, error) {
	var err error
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		// did we already resolve this value ?
		if ff := r.formFields[ref]; ff != nil {
			return ff, nil
		}
		// we haven't resolved it yet: do it
		o = r.resolve(ref)
	}
	if o == nil {
		return nil, nil
	}
	f, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("FormField", o)
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

	if da, ok := isString(f["DA"]); ok {
		fi.DA = da
	}

	if rect := rectangleFromArray(f.ArrayEntry("Rect")); rect != nil {
		fi.Rect = *rect
	}

	contents, _ := isString(f["Contents"])
	fi.Contents = decodeTextString(contents)

	fi.AP, err = r.resolveAppearanceDict(f["AP"])
	if err != nil {
		return nil, err
	}

	if isRef { // write back to the cache
		r.formFields[ref] = &fi
	}
	return &fi, nil
}

func rectangleFromArray(ar pdfcpu.Array) *model.Rectangle {
	if len(ar) < 4 {
		return nil
	}
	llx, _ := isNumber(ar[0])
	lly, _ := isNumber(ar[1])
	urx, _ := isNumber(ar[2])
	ury, _ := isNumber(ar[3])
	return &model.Rectangle{Llx: llx, Lly: lly, Urx: urx, Ury: ury}
}

func matrixFromArray(ar pdfcpu.Array) *model.Matrix {
	if len(ar) != 6 {
		return nil
	}
	var out model.Matrix
	for i := range out {
		out[i], _ = isNumber(ar[i])
	}
	return &out
}

func (r resolver) resolveAppearanceDict(o pdfcpu.Object) (*model.AppearanceDict, error) {
	ref, isRef := o.(pdfcpu.IndirectRef)
	if isRef {
		if ff := r.appearanceDicts[ref]; ff != nil {
			return ff, nil
		}
		o = r.resolve(ref)
	}
	if o == nil {
		return nil, nil
	}
	a, isDict := o.(pdfcpu.Dict)
	if !isDict {
		return nil, errType("AppearanceDict", o)
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
		obj = r.resolve(refApp)
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
					stream = r.resolve(refStream)
				}
				streamDict, ok := stream.(pdfcpu.StreamDict)
				if !ok {
					return nil, errType("Stream object", stream)
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
		return nil, errType("Appearance", obj)
	}
	return &out, nil
}

func (r resolver) processXObject(obj pdfcpu.StreamDict) (*model.XObject, error) {
	var ap model.XObject
	if rect := rectangleFromArray(obj.Dict.ArrayEntry("BBox")); rect != nil {
		ap.BBox = *rect
	}
	ap.Matrix = matrixFromArray(obj.Dict.ArrayEntry("Matrix"))
	if res := obj.Dict["Resources"]; res != nil {
		var err error
		ap.Resources, err = r.resolveOneResourceDict(res)
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

func (r resolver) processAcroForm(acroForm pdfcpu.Object) (model.AcroForm, error) {
	var out model.AcroForm
	if acroForm != nil {
		form, err := r.xref.DereferenceDict(acroForm)
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

// The value of this entry shall be a dictionary in which
// each key is a destination name and the corresponding value is either an array defining the destination, using
// the syntax shown in Table 151, or a dictionary with a D entry whose value is such an array.
func (r *resolver) resolveOneNamedDest(dest pdfcpu.Object) (*model.ExplicitDestination, error) {
	dest = r.resolve(dest)
	var (
		expDest *model.ExplicitDestination
		err     error
	)
	switch dest := dest.(type) {
	case pdfcpu.Array:
		expDest, err = r.resolveExplicitDestination(dest)
	case pdfcpu.Dict:
		D, isArray := dest["D"].(pdfcpu.Array)
		if !isArray {
			return nil, errType("(Dests value).D", dest["D"])
		}
		expDest, err = r.resolveExplicitDestination(D)
	default:
		return nil, errType("Dests value", dest)
	}
	return expDest, err
}

// the entry is a simple dictonnary
// In PDF 1.1, the correspondence between name objects and destinations shall be defined by the Dests entry in
// the document catalogue (see 7.7.2, “Document Catalog”).
func (r *resolver) processDictDests(entry pdfcpu.Object) (model.DestTree, error) {
	entry = r.resolve(entry)
	if entry == nil {
		return model.DestTree{}, nil
	}
	nameDict, isDict := entry.(pdfcpu.Dict)
	if !isDict {
		return model.DestTree{}, errType("Dests", entry)
	}
	var out model.DestTree
	for name, dest := range nameDict {
		expDest, err := r.resolveOneNamedDest(dest)
		if err != nil {
			return out, err
		}
		out.Names = append(out.Names, model.NameToDest{Name: name, Destination: expDest})
	}
	return out, nil
}

func (r *resolver) resolveDestTree(entry pdfcpu.Object) (*model.DestTree, error) {
	out := new(model.DestTree)
	err := r.resolveNameTree(entry, destNameTree{out: out})
	return out, err
}

func (r *resolver) resolveEmbeddedFilesTree(files pdfcpu.Object) (model.EmbeddedFileTree, error) {
	out := new(model.EmbeddedFileTree)
	err := r.resolveNameTree(files, embFileNameTree{out: out})
	return *out, err
}

func (r *resolver) processNameDict(entry pdfcpu.Object) (model.NameDictionnary, error) {
	var (
		out model.NameDictionnary
		err error
	)

	dict, _ := r.resolve(entry).(pdfcpu.Dict)
	if destsEntry := dict["Dests"]; destsEntry != nil {
		dests, err := r.resolveDestTree(destsEntry)
		if err != nil {
			return out, err
		}
		out.Dests = *dests
	}

	if embeddedFiles := dict["EmbeddedFiles"]; embeddedFiles != nil {
		out.EmbeddedFiles, err = r.resolveEmbeddedFilesTree(embeddedFiles)
		if err != nil {
			return out, err
		}
	}

	// TODO: other names
	return out, nil
}
