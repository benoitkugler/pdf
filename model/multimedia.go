package model

import (
	"fmt"
	"strings"
)

// LanguageArray contains pairs of strings. The first string in each pair shall be a language
// identifier. The second string is text associated with the language.
type LanguageArray [][2]string

func (l LanguageArray) Write(w PDFWritter, ref Reference) string {
	chunks := make([]string, 0, len(l)*2)
	for _, p := range l {
		chunks = append(chunks, p[0], p[1])
	}
	return writeStringsArray(chunks, w, TextString, ref)
}

func (l LanguageArray) Clone() LanguageArray {
	return append([][2]string(nil), l...)
}

// MediaCriteria
// All entry are optional
type MediaCriteria struct {
	A, C, O, S MaybeBool
	R          MaybeInt
	D          MediaBitDepth
	Z          MediaScreenSize
	V          []SoftwareIdentifier
	P          [2]Name
	L          []string
}

func (m MediaCriteria) Write(pdf PDFWritter, context Reference) string {
	b := newBuffer()
	if bo, ok := m.A.(ObjBool); ok {
		b.fmt("/A %v", bo)
	}
	if bo, ok := m.C.(ObjBool); ok {
		b.fmt("/C %v", bo)
	}
	if bo, ok := m.O.(ObjBool); ok {
		b.fmt("/O %v", bo)
	}
	if bo, ok := m.S.(ObjBool); ok {
		b.fmt("/S %v", bo)
	}
	if i, ok := m.R.(ObjInt); ok {
		b.fmt("/R %d", i)
	}
	if m.D != (MediaBitDepth{}) {
		b.WriteString("/D " + m.D.Write())
	}
	if m.Z != (MediaScreenSize{}) {
		b.WriteString("/Z " + m.Z.Write())
	}
	if len(m.V) != 0 {
		chunks := make([]string, len(m.V))
		for i, s := range m.V {
			chunks[i] = s.Write(pdf, context)
		}
		b.fmt("/V [%s]", strings.Join(chunks, " "))
	}
	var ps []Name
	if m.P[0] != "" {
		ps = append(ps, m.P[0])
		if m.P[1] != "" {
			ps = append(ps, m.P[1])
		}
	}
	if len(ps) != 0 {
		b.WriteString("/P " + writeNameArray(ps))
	}
	if len(m.L) != 0 {
		b.WriteString("/L " + writeStringsArray(m.L, pdf, TextString, context))
	}
	b.line(">>")
	return b.String()
}

func (m *MediaCriteria) Clone() *MediaCriteria {
	if m == nil {
		return nil
	}
	out := *m
	if m.V != nil {
		out.V = make([]SoftwareIdentifier, len(m.V))
		for i, s := range m.V {
			out.V[i] = s.Clone()
		}
	}
	out.L = append([]string(nil), m.L...)
	return &out
}

type MediaBitDepth struct {
	V int // required
	M int // optional
}

func (m MediaBitDepth) Write() string {
	return fmt.Sprintf("<<∕V %d /M %d>>", m.V, m.M)
}

type MediaScreenSize struct {
	V [2]int // required
	M int    // optional
}

func (m MediaScreenSize) Write() string {
	return fmt.Sprintf("<<∕V %s /M %d>>", writeIntArray(m.V[:]), m.M)
}

// Table 292 – Entries in a software identifier dictionary
type SoftwareIdentifier struct {
	U      string // required, ASCII string
	L, H   []int
	LI, HI MaybeBool
	OS     []string
}

func (s SoftwareIdentifier) Write(pdf PDFWritter, context Reference) string {
	out := "<</U " + EscapeByteString([]byte(s.U))
	if len(s.L) != 0 {
		out += "/L " + writeIntArray(s.L)
	}
	if len(s.H) != 0 {
		out += "/H " + writeIntArray(s.H)
	}
	if b, ok := s.LI.(ObjBool); ok {
		out += fmt.Sprintf("/LI %v", b)
	}
	if b, ok := s.HI.(ObjBool); ok {
		out += fmt.Sprintf("/HI %v", b)
	}
	if len(s.OS) != 0 {
		out += "/OS " + writeStringsArray(s.OS, pdf, ByteString, context)
	}
	out += ">>"
	return out
}

func (s SoftwareIdentifier) Clone() SoftwareIdentifier {
	out := s
	out.L = append([]int(nil), s.L...)
	out.H = append([]int(nil), s.H...)
	out.OS = append([]string(nil), s.OS...)
	return out
}

// RenditionDict is either a media rendition or a selector rendition
// See 13.2.3 - Renditions
type RenditionDict struct {
	N       string // optional
	Subtype Rendition
	MH, BE  *MediaCriteria // optional, written in PDF as a dict with a C entry
}

func (rd RenditionDict) pdfString(w pdfWriter, r Reference) string {
	out := "<<"
	if rd.Subtype != nil {
		out += rd.Subtype.renditionFields(w, r)
	}
	if rd.N != "" {
		out += "/N " + w.EncodeString(rd.N, TextString, r)
	}
	if rd.MH != nil {
		out += fmt.Sprintf("/MH <</C %s>>", rd.MH.Write(w, r))
	}
	if rd.BE != nil {
		out += fmt.Sprintf("/BE <</C %s>>", rd.BE.Write(w, r))
	}
	out += ">>"
	return out
}

func (rd RenditionDict) clone(cache cloneCache) RenditionDict {
	out := rd
	if rd.Subtype != nil {
		out.Subtype = rd.Subtype.clone(cache)
	}
	out.MH = rd.MH.Clone()
	out.BE = rd.BE.Clone()
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
	out.SP = rm.SP.Clone()
	return out
}

func (rm RenditionMedia) renditionFields(w pdfWriter, r Reference) string {
	out := "/S/MR/C " + rm.C.pdfString(w, r) + "/P " + rm.P.Write()
	if rm.SP != (MediaScreen{}) {
		out += "/SP " + rm.SP.Write(w, r)
	}
	return out
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
		fields += "/Alt " + ms.Alt.Write(pdf, ref)
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
		fields += "/Alt " + ms.Alt.Write(pdf, ref)
	}
	if !ms.MH.IsEmpty() {
		fields += "/MH " + ms.MH.Write(pdf, ref)
	}
	if !ms.BE.IsEmpty() {
		fields += "/BE " + ms.BE.Write(pdf, ref)
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

func (m MediaClipSectionLimits) Write(pdf PDFWritter, ref Reference) string {
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

func (m MediaPlayer) Write() string {
	out := "<<"
	if m.MH != (MediaPlayerParameters{}) {
		out += "/MH " + m.MH.Write()
	}
	if m.BE != (MediaPlayerParameters{}) {
		out += "/BE " + m.BE.Write()
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

func (m MediaPlayerParameters) Write() string {
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
	return fmt.Sprintf("<</S/T /T<</S/S/V %f>> >>", f)
}

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
	return fmt.Sprintf("<</S/T/T<</S/S/V %f>>>>", f)
}

func (m ObjStringLiteral) mediaOffsetString(w PDFWritter, ref Reference) string {
	return fmt.Sprintf("<</S/M/M %s>>", w.EncodeString(string(m), TextString, ref))
}

type MediaScreen struct {
	MH, BE *MediaScreenParams // optional
}

func (m MediaScreen) Write(w PDFWritter, r Reference) string {
	out := "<<"
	if m.MH != nil {
		out += "/MH " + m.MH.Write(w, r)
	}
	if m.BE != nil {
		out += "/BE " + m.BE.Write(w, r)
	}
	out += ">>"
	return out
}

func (m MediaScreen) Clone() MediaScreen {
	out := MediaScreen{
		MH: m.MH.Clone(),
		BE: m.BE.Clone(),
	}
	return out
}

type MediaScreenParams struct {
	W MaybeInt
	B *[3]Fl // optional
	O MaybeFloat
	M int
	F *MediaScreenFloatingWindow // optional
}

func (m MediaScreenParams) Write(w PDFWritter, r Reference) string {
	out := "<<"
	if w, ok := m.W.(ObjInt); ok {
		out += fmt.Sprintf("/W %d", w)
	}
	if m.B != nil {
		out += "/B " + writeFloatArray((*m.B)[:])
	}
	if o, ok := m.O.(ObjFloat); ok {
		out += fmt.Sprintf("/O %f", o)
	}
	if m.F != nil {
		out += "/F " + m.F.Write(w, r)
	}
	out += ">>"
	return out
}

func (m *MediaScreenParams) Clone() *MediaScreenParams {
	if m == nil {
		return nil
	}
	out := *m
	if m.B != nil {
		tmp := *m.B
		out.B = &tmp
	}
	out.F = m.F.Clone()
	return &out
}

type MediaScreenFloatingWindow struct {
	D     [2]int // required, width and height
	RT, R uint8
	P, O  MaybeInt
	T, UC MaybeBool
	TT    LanguageArray
}

func (m MediaScreenFloatingWindow) Write(w PDFWritter, r Reference) string {
	b := newBuffer()
	b.WriteString("<</D " + writeIntArray(m.D[:]))
	if m.RT != 0 {
		b.fmt("/RT %d", m.RT)
	}
	if m.R != 0 {
		b.fmt("/R %d", m.R)
	}
	if i, ok := m.P.(ObjInt); ok {
		b.fmt("/P %d", i)
	}
	if i, ok := m.O.(ObjInt); ok {
		b.fmt("/O %d", i)
	}
	if bo, ok := m.T.(ObjBool); ok {
		b.fmt("/T %v", bo)
	}
	if bo, ok := m.UC.(ObjBool); ok {
		b.fmt("/UC %v", bo)
	}
	if len(m.TT) != 0 {
		b.WriteString("/TT " + m.TT.Write(w, r))
	}
	b.WriteString(">>")
	return b.String()
}

func (m *MediaScreenFloatingWindow) Clone() *MediaScreenFloatingWindow {
	if m == nil {
		return m
	}
	out := *m
	out.TT = m.TT.Clone()
	return &out
}
