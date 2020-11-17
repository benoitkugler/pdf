package model

import (
	"fmt"
	"sort"
	"strings"
)

// generic code for limits

type numTree interface {
	nums() []int
	kids() []numTree
	Limits() [2]int
}

func limitsNum(n numTree) [2]int {
	kids := n.kids()
	if len(kids) == 0 { // leaf node
		nums := n.nums()
		if len(nums) == 0 {
			return [2]int{}
		}
		sort.Ints(nums)
		return [2]int{nums[0], nums[len(nums)-1]}
	}
	limits := kids[0].Limits()
	for _, kid := range kids[1:] {
		kidLimits := kid.Limits()
		if kidLimits[0] < limits[0] {
			limits[0] = kidLimits[0]
		}
		if kidLimits[1] > limits[1] {
			limits[1] = kidLimits[1]
		}
	}
	return limits
}

type nameTree interface {
	names() []string
	kids() []nameTree
	Limits() [2]string
}

func limitsName(n nameTree) [2]string {
	kids := n.kids()
	if len(kids) == 0 { // leaf node
		names := n.names()
		if len(names) == 0 {
			return [2]string{}
		}
		sort.Strings(names)
		return [2]string{names[0], names[len(names)-1]}
	}
	limits := kids[0].Limits()
	for _, kid := range kids[1:] {
		kidLimits := kid.Limits()
		if kidLimits[0] < limits[0] {
			limits[0] = kidLimits[0]
		}
		if kidLimits[1] > limits[1] {
			limits[1] = kidLimits[1]
		}
	}
	return limits
}

// NameToDest associate an explicit destination
// to a name.
type NameToDest struct {
	Name        string
	Destination *DestinationExplicit // indirect object
}

func (n NameToDest) clone(cache cloneCache) NameToDest {
	out := n
	if n.Destination != nil {
		dest := n.Destination.clone(cache).(DestinationExplicit)
		out.Destination = &dest
	}
	return out
}

// DestTree links a serie of arbitrary name
// to explicit destination, enabling `NamedDestination`
// to reference them
type DestTree struct {
	Kids  []DestTree
	Names []NameToDest
}

func (d DestTree) names() []string {
	out := make([]string, len(d.Names))
	for i, k := range d.Names {
		out[i] = k.Name
	}
	return out
}

func (d DestTree) kids() []nameTree {
	out := make([]nameTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

// Limits specify the (lexically) least and greatest keys included in the Names array of
// a leaf node or in the Names arrays of any leaf nodes that are descendants of an
// intermediate node.
func (d DestTree) Limits() [2]string {
	return limitsName(d)
}

// LookupTable walks the name tree and
// accumulates the result into one map
func (d DestTree) LookupTable() map[string]*DestinationExplicit {
	out := make(map[string]*DestinationExplicit)
	for _, v := range d.Names {
		out[v.Name] = v.Destination
	}
	for _, kid := range d.Kids {
		for name, dest := range kid.LookupTable() {
			out[name] = dest
		}
	}
	return out
}

func (p DestTree) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	limits := p.Limits()
	b.line("<</Limits [%s %s]",
		pdf.EncodeString(limits[0], ByteString, ref), pdf.EncodeString(limits[1], ByteString, ref))
	if len(p.Kids) != 0 {
		b.fmt("/Kids [")
		for _, kid := range p.Kids {
			kidRef := pdf.createObject()
			pdf.writeObject(kid.pdfString(pdf, kidRef), nil, kidRef)
			b.fmt("%s ", kidRef)
		}
		b.line("]")
	}
	if len(p.Names) != 0 {
		b.fmt("/Names [ ")
		for _, name := range p.Names {
			b.fmt("%s %s ", pdf.EncodeString(name.Name, ByteString, ref), name.Destination.pdfDestination(pdf))
		}
		b.line("]")
	}
	b.fmt(">>")
	return b.String()
}

// cache.pages must have been filled
func (d DestTree) clone(cache cloneCache) DestTree {
	out := d
	if d.Kids != nil { // preserve reflect.DeepEqual
		out.Kids = make([]DestTree, len(d.Kids))
	}
	for i, k := range d.Kids {
		out.Kids[i] = k.clone(cache)
	}
	if d.Names != nil { // preserve reflect.DeepEqual
		out.Names = make([]NameToDest, len(d.Names))
	}
	for i, k := range d.Names {
		out.Names[i] = k.clone(cache)
	}
	return out
}

// ----------------------------------------------------------------------

type NameToFile struct {
	Name     string
	FileSpec *FileSpec // indirect object
}

func (f NameToFile) clone(cache cloneCache) NameToFile {
	out := f
	out.FileSpec = cache.checkOrClone(f.FileSpec).(*FileSpec)
	return out
}

// EmbeddedFileTree is written as a Name Tree in PDF,
// but, since it generally won't be big, is
// represented here as a flat list.
type EmbeddedFileTree []NameToFile

func (d EmbeddedFileTree) names() []string {
	out := make([]string, len(d))
	for i, k := range d {
		out[i] = k.Name
	}
	return out
}

func (d EmbeddedFileTree) kids() []nameTree {
	return nil
}

func (efs EmbeddedFileTree) Limits() [2]string {
	return limitsName(efs)
}

func (p EmbeddedFileTree) pdfString(pdf pdfWriter, ref Reference) string {
	lims := p.Limits()
	chunks := make([]string, len(p))
	for i, f := range p {
		fsRef := pdf.addItem(f.FileSpec)
		chunks[i] = fmt.Sprintf("%s %s",
			pdf.EncodeString(f.Name, ByteString, ref), fsRef)
	}
	return fmt.Sprintf("<</Limits [%s %s] /Names [%s]>>",
		pdf.EncodeString(lims[0], ByteString, ref), pdf.EncodeString(lims[1], ByteString, ref),
		strings.Join(chunks, " "))
}

func (p EmbeddedFileTree) clone(cache cloneCache) EmbeddedFileTree {
	if p == nil { // preserve reflect.DeepEqual
		return p
	}
	out := make(EmbeddedFileTree, len(p))
	for i, f := range p {
		out[i] = f.clone(cache)
	}
	return out
}

// -----------------------------------------------------------------------

// PageLabel defines the labelling characteristics for the pages
// in a range.
type PageLabel struct {
	S  Name
	P  string // optional
	St int    // optionnal default to 1
}

func (p PageLabel) pdfString(st PDFStringEncoder, ref Reference) string {
	return fmt.Sprintf("<</S %s /P %s /St %d>>", p.S, st.EncodeString(p.P, TextString, ref), p.St)
}

type NumToPageLabel struct {
	Num       int
	PageLabel PageLabel // rather a direct object
}

// return two elements, to be included in an array
func (n NumToPageLabel) pdfString(pdf pdfWriter, ref Reference) string {
	return fmt.Sprintf("%d %s", n.Num, n.PageLabel.pdfString(pdf, ref))
}

type PageLabelsTree struct {
	Kids []PageLabelsTree
	Nums []NumToPageLabel
}

func (d PageLabelsTree) nums() []int {
	out := make([]int, len(d.Nums))
	for i, k := range d.Nums {
		out[i] = k.Num
	}
	return out
}

func (d PageLabelsTree) kids() []numTree {
	out := make([]numTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

// Limits specify the (numerically) least and greatest keys included in
// the Nums array of a leaf node or in the Nums arrays of any leaf nodes that are
// descendants of an intermediate node.
func (d PageLabelsTree) Limits() [2]int {
	return limitsNum(d)
}

// LookupTable walks the number tree and
// accumulates the result into one map
func (d PageLabelsTree) LookupTable() map[int]PageLabel {
	out := make(map[int]PageLabel)
	for _, v := range d.Nums {
		out[v.Num] = v.PageLabel
	}
	for _, kid := range d.Kids {
		for name, dest := range kid.LookupTable() {
			out[name] = dest
		}
	}
	return out
}

func (p PageLabelsTree) pdfString(pdf pdfWriter, ref Reference) string {
	b := newBuffer()
	limits := p.Limits()
	b.line("<</Limits [%d %d]", limits[0], limits[1])
	if len(p.Kids) != 0 {
		b.fmt("/Kids [")
		for _, kid := range p.Kids {
			kidRef := pdf.createObject()
			pdf.writeObject(kid.pdfString(pdf, kidRef), nil, kidRef)
			b.fmt("%s ", kidRef)
		}
		b.line("]")
	}
	if len(p.Nums) != 0 {
		b.fmt("/Nums [ ")
		for _, num := range p.Nums {
			b.fmt("%s ", num.pdfString(pdf, ref))
		}
		b.line("]")
	}
	b.fmt(">>")
	return b.String()
}

func (p PageLabelsTree) Clone() PageLabelsTree {
	out := p
	out.Nums = append([]NumToPageLabel(nil), p.Nums...)
	if p.Kids != nil {
		out.Kids = make([]PageLabelsTree, len(p.Kids))
		for i, k := range p.Kids {
			out.Kids[i] = k.Clone()
		}
	}
	return out
}

// ------------------------------------------------------------
type NameToStructureElement struct {
	Name      string
	Structure *StructureElement
}

// return two elements, to be included in an array
func (n NameToStructureElement) pdfString(pdf pdfWriter) string {
	return fmt.Sprintf("%s %s", n.Name, pdf.structure[n.Structure])
}

func (n NameToStructureElement) clone(cache cloneCache) NameToStructureElement {
	out := n
	out.Structure = cache.structure[n.Structure]
	return out
}

type IDTree struct {
	Kids  []IDTree
	Names []NameToStructureElement
}

func (d IDTree) names() []string {
	out := make([]string, len(d.Names))
	for i, k := range d.Names {
		out[i] = k.Name
	}
	return out
}

func (d IDTree) kids() []nameTree {
	out := make([]nameTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

// Limits specify the (lexically) least and greatest keys included in the Names array of
// a leaf node or in the Names arrays of any leaf nodes that are descendants of an
// intermediate node.
func (d IDTree) Limits() [2]string {
	return limitsName(d)
}

// requires that the structure tree is already clone
func (d IDTree) clone(cache cloneCache) IDTree {
	out := d
	if d.Kids != nil { // preserve reflect.DeepEqual
		out.Kids = make([]IDTree, len(d.Kids))
		for i, k := range d.Kids {
			out.Kids[i] = k.clone(cache)
		}
	}
	if d.Names != nil { // preserve reflect.DeepEqual
		out.Names = make([]NameToStructureElement, len(d.Names))
		for i, k := range d.Names {
			out.Names[i] = k.clone(cache)
		}
	}
	return out
}

// requires the structure to have been written
func (d IDTree) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	limits := d.Limits()
	b.line("<</Limits [%s %s]", limits[0], limits[1])
	if len(d.Kids) != 0 {
		b.fmt("/Kids [")
		for _, kid := range d.Kids {
			kidRef := pdf.addObject(kid.pdfString(pdf), nil)
			b.fmt("%s ", kidRef)
		}
		b.line("]")
	}
	if len(d.Names) != 0 {
		b.fmt("/Names [ ")
		for _, num := range d.Names {
			b.fmt("%s ", num.pdfString(pdf))
		}
		b.line("]")
	}
	b.fmt(">>")
	return b.String()
}

type NumToParent struct {
	Num int
	// either Parent or Parents must be non nil
	Parent  *StructureElement
	Parents []*StructureElement
}

func (n NumToParent) pdfString(pdf pdfWriter) string {
	var parent string
	if n.Parent != nil {
		parent = pdf.structure[n.Parent].String()
	} else {
		refs := make([]Reference, len(n.Parents))
		for i, p := range n.Parents {
			refs[i] = pdf.structure[p]
		}
		parent = writeRefArray(refs)
	}
	return fmt.Sprintf("%d %s", n.Num, parent)
}

func (n NumToParent) clone(cache cloneCache) NumToParent {
	out := n
	out.Parent = cache.structure[n.Parent]
	if n.Parents != nil {
		out.Parents = make([]*StructureElement, len(n.Parents))
		for i, p := range n.Parents {
			out.Parents[i] = cache.structure[p]
		}
	}
	return out
}

type ParentTree struct {
	Kids []ParentTree
	Nums []NumToParent
}

func (d ParentTree) nums() []int {
	out := make([]int, len(d.Nums))
	for i, k := range d.Nums {
		out[i] = k.Num
	}
	return out
}

func (d ParentTree) kids() []numTree {
	out := make([]numTree, len(d.Kids))
	for i, k := range d.Kids {
		out[i] = k
	}
	return out
}

func (d ParentTree) Limits() [2]int {
	return limitsNum(d)
}

// structure elements must have been cloned
func (d ParentTree) clone(cache cloneCache) ParentTree {
	out := d
	if d.Nums != nil { // preserve nil
		out.Nums = make([]NumToParent, len(d.Nums))
		for i, n := range d.Nums {
			out.Nums[i] = n.clone(cache)
		}
	}
	if d.Kids != nil { // preserve nil
		out.Kids = make([]ParentTree, len(d.Kids))
		for i, k := range d.Kids {
			out.Kids[i] = k.clone(cache)
		}
	}
	return out
}

func (d ParentTree) pdfString(pdf pdfWriter) string {
	b := newBuffer()
	limits := d.Limits()
	b.line("<</Limits [%d %d]", limits[0], limits[1])
	if len(d.Kids) != 0 {
		b.fmt("/Kids [")
		for _, kid := range d.Kids {
			kidRef := pdf.addObject(kid.pdfString(pdf), nil)
			b.fmt("%s ", kidRef)
		}
		b.line("]")
	}
	if len(d.Nums) != 0 {
		b.fmt("/Nums [ ")
		for _, num := range d.Nums {
			b.fmt("%s ", num.pdfString(pdf))
		}
		b.line("]")
	}
	b.fmt(">>")
	return b.String()
}
