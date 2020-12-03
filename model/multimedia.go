package model

import (
	"fmt"
	"strings"
)

// LanguageArray contains pairs of strings. The first string in each pair shall be a language
// identifier. The second string is text associated with the language.
type LanguageArray [][2]string

func (l LanguageArray) PDFString(w PDFWritter, ref Reference) string {
	chunks := make([]string, 0, len(l)*2)
	for _, p := range l {
		chunks = append(chunks, p[0], p[1])
	}
	return writeStringsArray(chunks, w, TextString, ref)
}

func (l LanguageArray) Clone() LanguageArray {
	return append([][2]string(nil), l...)
}

// RenditionDict is either a media rendition or a selector rendition
// See 13.2.3 - Renditions
type RenditionDict struct {
	N       string // optional
	Subtype Rendition
}

func (rd RenditionDict) pdfString(w pdfWriter, r Reference) string {
	out := "<<"
	if rd.Subtype != nil {
		out += rd.Subtype.renditionFields(w, r)
	}
	if rd.N != "" {
		out += "/N " + w.EncodeString(rd.N, TextString, r)
	}
	out += ">>"
	return out
}

func (rd RenditionDict) clone(cache cloneCache) RenditionDict {
	out := rd
	if rd.Subtype != nil {
		out.Subtype = rd.Subtype.clone(cache)
	}
	return out
}

// Rendition is either a selector or a media rendition
type Rendition interface {
	renditionFields(pdfWriter, Reference) string
	clone(cloneCache) Rendition
}

type RenditionSelector struct {
	R []RenditionDict
}

func (rs RenditionSelector) clone(cache cloneCache) Rendition {
	if rs.R == nil {
		return RenditionSelector{}
	}
	out := make([]RenditionDict, len(rs.R))
	for i, r := range rs.R {
		out[i] = r.clone(cache)
	}
	return RenditionSelector{R: out}
}

func (rs RenditionSelector) renditionFields(w pdfWriter, ref Reference) string {
	chunks := make([]string, len(rs.R))
	for i, rd := range rs.R {
		chunks[i] = rd.pdfString(w, ref)
	}
	return "/S/SR/R [" + strings.Join(chunks, " ") + "]"
}

type RenditionMedia struct {
	C  MediaClipDict
	P  MediaPlayer
	SP MediaScreen
}

func (rm RenditionMedia) clone(cache cloneCache) Rendition {
	out := rm
	out.C = rm.C.clone(cache)
	// TODO: SP
	return out
}

func (rm RenditionMedia) renditionFields(w pdfWriter, r Reference) string {
	// TODO: SP
	return "/S/MR/C " + rm.C.pdfString(w, r) + "/P " + rm.P.PDFString()
}

type MediaClipDict struct {
	N       string    // optional
	Subtype MediaClip // required
}

func (mc MediaClipDict) pdfString(w pdfWriter, ref Reference) string {
	fields := mc.Subtype.mediaClipFields(w, ref)
	if mc.N != "" {
		fields += "/N " + w.EncodeString(mc.N, TextString, ref)
	}
	return "<<" + fields + ">>"
}

func (mc MediaClipDict) clone(cache cloneCache) MediaClipDict {
	out := mc
	if mc.Subtype != nil {
		out.Subtype = mc.Subtype.clone(cache)
	}
	return out
}

// MediaClip is either a Media Clip Data or a Media Clip Section
type MediaClip interface {
	mediaClipFields(pdf pdfWriter, ref Reference) string
	clone(cache cloneCache) MediaClip
}

type MediaClipPermission string

const (
	TempNever   MediaClipPermission = "TEMPNEVER"
	TempExtract MediaClipPermission = "TEMPEXTRACT"
	TempAccess  MediaClipPermission = "TEMPACCESS"
	TempAlways  MediaClipPermission = "TEMPALWAYS"
)

type MediaClipData struct {
	D   MediaClipDataContent // required
	CT  string               // optional, MIME, must be ASCII
	P   MediaClipPermission  // optional, default to  TempNever
	Alt LanguageArray
	// TODO: add PL field

	MH, BE string // optional, ASCII url, written as a dict in PDF
}

func (m MediaClipData) clone(cache cloneCache) MediaClip {
	out := m
	out.Alt = m.Alt.Clone()
	if m.D != nil {
		out.D = cache.checkOrClone(m.D).(MediaClipDataContent)
	}
	return out
}

func (ms MediaClipData) mediaClipFields(pdf pdfWriter, ref Reference) string {
	fields := "/S/MCD"
	if ms.D != nil {
		ref := pdf.addItem(ms.D)
		fields += "/D " + ref.String()
	}
	if ms.CT != "" {
		fields += "/CT " + pdf.EncodeString(ms.CT, ByteString, ref)
	}
	if ms.P != "" {
		fields += fmt.Sprintf("/P <</TD %s>>", pdf.EncodeString(string(ms.P), ByteString, ref))
	}
	if len(ms.Alt) != 0 {
		fields += "/Alt " + ms.Alt.PDFString(pdf, ref)
	}
	if ms.MH != "" {
		fields += "/MH <</BU %s>>" + pdf.EncodeString(ms.MH, ByteString, ref)
	}
	if ms.BE != "" {
		fields += "/BE <</BU %s>>" + pdf.EncodeString(ms.BE, ByteString, ref)
	}
	return fields
}

// MediaClipDataContent is either a file specification
// or an XForm object
type MediaClipDataContent interface {
	Referenceable
	isMCDContent()
}

func (dc *XObjectForm) isMCDContent() {}

func (dc *FileSpec) isMCDContent() {}

// MediaClipSection defines a continuous section of another media clip object.
// See Table 277 – Additional entries in a media clip section dictionary
type MediaClipSection struct {
	D      MediaClipDict          // required
	Alt    LanguageArray          // optional
	MH, BE MediaClipSectionLimits // optional
}

func (ms MediaClipSection) mediaClipFields(pdf pdfWriter, ref Reference) string {
	fields := "/S/MCS/D " + ms.D.pdfString(pdf, ref)
	if len(ms.Alt) != 0 {
		fields += "/Alt " + ms.Alt.PDFString(pdf, ref)
	}
	if !ms.MH.IsEmpty() {
		fields += "/MH " + ms.MH.PDFString(pdf, ref)
	}
	if !ms.BE.IsEmpty() {
		fields += "/BE " + ms.BE.PDFString(pdf, ref)
	}
	return fields
}

// clone returns a deep copy with concrete type `MediaClipSection`.
func (ms MediaClipSection) clone(cache cloneCache) MediaClip {
	out := ms
	out.D = ms.D.clone(cache)
	out.Alt = ms.Alt.Clone()
	return out
}

type MediaClipSectionLimits struct {
	B MediaOffset // optional (begining)
	E MediaOffset // optional (end)
}

// IsEmpty return true if the dict is empty.
func (m MediaClipSectionLimits) IsEmpty() bool {
	return m.B == nil && m.E == nil
}

func (m MediaClipSectionLimits) PDFString(pdf PDFWritter, ref Reference) string {
	out := "<<"
	if m.B != nil {
		out += "/B " + m.B.mediaOffsetString(pdf, ref)
	}
	if m.E != nil {
		out += "/E " + m.E.mediaOffsetString(pdf, ref)
	}
	out += ">>"
	return out
}

type MediaPlayer struct {
	// TODO: PL
	MH, BE MediaPlayerParameters // optional
}

func (m MediaPlayer) PDFString() string {
	out := "<<"
	if m.MH != (MediaPlayerParameters{}) {
		out += "/MH " + m.MH.PDFString()
	}
	if m.BE != (MediaPlayerParameters{}) {
		out += "/BE " + m.BE.PDFString()
	}
	out += ">>"
	return out
}

type MediaPlayerParameters struct {
	V       MaybeInt
	C       bool
	F       MaybeInt      // optional 0 to 5
	D       MediaDuration // optional
	NotAuto bool          // optional, default to false, written in PDF as /A
	RC      MaybeFloat    // optional, default to 1, 0 means for repeat for ever
}

func (m MediaPlayerParameters) PDFString() string {
	out := "<<"
	if m.V != nil {
		out += fmt.Sprintf("/V %d", m.V.(ObjInt))
	}
	out += fmt.Sprintf("/C %v", m.C)
	if m.F != nil {
		out += fmt.Sprintf("/F %d", m.F.(ObjInt))
	}
	if m.D != nil {
		out += "/D " + m.D.mediaDurationString()
	}
	out += fmt.Sprintf("/A %v", !m.NotAuto)
	if m.RC != nil {
		out += fmt.Sprintf("/RC %.2f", m.RC.(ObjFloat))
	}
	out += ">>"
	return out
}

// MediaDuration is either a name I or F (Name) or a time span (ObjFloat)
type MediaDuration interface {
	mediaDurationString() string
}

var (
	MediaDurationIntrinsic MediaDuration = Name("I")
	MediaDurationInfinite  MediaDuration = Name("F")
)

func (n Name) mediaDurationString() string {
	return fmt.Sprintf("<</S %s>>", n)
}

func (f ObjFloat) mediaDurationString() string {
	return fmt.Sprintf("<</S/T /T<</S/S/V %.3f>> >>", f)
}

type MediaScreen struct{}

// MediaOffset specifies an offset into a media object. It is either:
// 	- a time, in seconds as ObjFloat
//	- a frame, as ObjInt
// 	- a marker, as ObjStringLitteral
type MediaOffset interface {
	mediaOffsetString(PDFWritter, Reference) string
}

func (i ObjInt) mediaOffsetString(PDFWritter, Reference) string {
	return fmt.Sprintf("<</S/F/F %d>>", i)
}

func (f ObjFloat) mediaOffsetString(PDFWritter, Reference) string {
	return fmt.Sprintf("<</S/T/T<</S/S/V %.3f>>>>", f)
}

func (m ObjStringLiteral) mediaOffsetString(w PDFWritter, ref Reference) string {
	return fmt.Sprintf("<</S/M/M %s>>", w.EncodeString(string(m), TextString, ref))
}
