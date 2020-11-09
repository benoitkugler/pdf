package writer

import "github.com/benoitkugler/pdf/model"

func (w *writer) writeNameDests(model.DestTree) ref                 { return 0 }
func (w *writer) writeNameEmbeddedFiles(model.EmbeddedFileTree) ref { return 0 }
func (w *writer) writeDests(model.DestTree) ref                     { return 0 }
func (w *writer) writeViewerPref(model.ViewerPreferences) ref       { return 0 }
func (w *writer) writeAcroForm(model.AcroForm) ref                  { return 0 }
