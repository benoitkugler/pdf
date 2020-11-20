package reader

import (
	"log"

	"github.com/benoitkugler/pdf/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

func (r resolver) processAction(ac pdfcpu.Object) (out model.Action, err error) {
	action, _ := r.resolve(ac).(pdfcpu.Dict)
	if action["S"] == nil {
		return
	}
	name, _ := r.resolveName(action["S"])
	switch name {
	case "URI":
		var subac model.ActionURI
		subac.URI, _ = isString(r.resolve(action["URI"]))
		subac.IsMap, _ = r.resolveBool(action["IsMap"])
		out.ActionType = subac
	case "GoTo":
		dest, err := r.processDestination(action["D"])
		if err != nil {
			return out, err
		}
		out.ActionType = model.ActionGoTo{D: dest}
	case "GoToR":
		dest, err := r.processDestination(action["D"])
		if err != nil {
			return out, err
		}
		subac := model.ActionRemoteGoTo{D: dest}
		subac.NewWindow, _ = r.resolveBool(action["NewWindow"])
		subac.F, err = r.resolveFileSpec(action["F"])
		out.ActionType = subac
	case "Launch":
		subac := model.ActionRemoteGoTo{}
		subac.NewWindow, _ = r.resolveBool(action["NewWindow"])
		subac.F, err = r.resolveFileSpec(action["F"])
		out.ActionType = subac
	case "GoToE":
		dest, err := r.processDestination(action["D"])
		if err != nil {
			return out, err
		}
		subac := model.ActionEmbeddedGoTo{D: dest}
		subac.NewWindow, _ = r.resolveBool(action["NewWindow"])
		if action["F"] != nil {
			subac.F, err = r.resolveFileSpec(action["F"])
			if err != nil {
				return out, err
			}
		}
		subac.T, err = r.resolveEmbeddedTarget(action["T"])
		out.ActionType = subac
	case "Hide":
		var subac model.ActionHide
		if hide, ok := r.resolveBool(action["H"]); ok { // false is not the default value
			subac.Show = !hide
		}
		if array, isArray := r.resolveArray(action["T"]); isArray { // many targets
			subac.T = make([]model.ActionHideTarget, len(array))
			for i, t := range array {
				subac.T[i], err = r.resolveOneHideTarget(t)
				if err != nil {
					return out, err
				}
			}
		} else { // one target
			t, err := r.resolveOneHideTarget(action["T"])
			if err != nil {
				return out, err
			}
			subac.T = []model.ActionHideTarget{t}
		}
		out.ActionType = subac
	case "Named":
		n, _ := r.resolveName(action["N"])
		out.ActionType = model.ActionNamed(n)
	case "JavaScript":
		var js string
		if K, ok := r.resolve(action).(pdfcpu.Dict); ok {
			js = r.textOrStream(K["JS"])
		}
		out.ActionType = model.ActionJavaScript{JS: js}
	default:
		log.Println("unsupported action:", name)
		return out, nil
	}

	if arr, isArray := r.resolveArray(action["Next"]); isArray { // many next actions
		out.Next = make([]model.Action, len(arr))
		for i, n := range arr {
			out.Next[i], err = r.processAction(n)
			if err != nil {
				return out, err
			}
		}
	} else { // maybe one next, try it
		next, err := r.processAction(action["Next"])
		if err != nil {
			return out, err
		}
		if next.ActionType != nil {
			out.Next = []model.Action{next}
		}
	}
	return out, nil
}

func (r resolver) resolveOneHideTarget(o pdfcpu.Object) (model.ActionHideTarget, error) {
	if st, is := isString(r.resolve(o)); is { // text string
		return model.HideTargetFormName(decodeTextString(st)), nil
	}
	return r.resolveAnnotation(o)
}

func (r resolver) resolveEmbeddedTarget(o pdfcpu.Object) (out *model.EmbeddedTarget, err error) {
	o = r.resolve(o)
	if o == nil {
		return nil, nil
	}
	dict, ok := o.(pdfcpu.Dict)
	if !ok {
		return nil, errType("Target dictionary", o)
	}
	out = new(model.EmbeddedTarget)
	out.R, _ = r.resolveName(dict["R"])
	out.N, _ = isString(r.resolve(dict["N"]))
	P := r.resolve(dict["P"])
	if p, ok := isString(P); ok {
		out.P = model.EmbeddedTargetDestNamed(p)
	} else if p, ok := r.resolveInt(P); ok {
		out.P = model.EmbeddedTargetDestPage(p)
	}
	A := r.resolve(dict["A"])
	if a, ok := isString(A); ok {
		out.A = model.EmbeddedTargetAnnotNamed(a)
	} else if a, ok := r.resolveInt(P); ok {
		out.A = model.EmbeddedTargetAnnotIndex(a)
	}
	out.T, err = r.resolveEmbeddedTarget(o)
	return out, err
}
