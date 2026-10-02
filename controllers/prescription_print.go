package controllers

import (
	"html/template"
	"io"

	"github.com/vetflow/veterinary-api/models"
)

// PrintablePrescription is the aggregated, display-ready view of a prescription
// used by the print endpoint. It embeds the raw prescription and adds the
// resolved human-readable header (pet, tutor and veterinarian) so neither the
// HTML template nor a JSON client has to perform its own joins.
type PrintablePrescription struct {
	*models.Prescription
	PetName    string `json:"pet_name"`
	PetSpecies string `json:"pet_species"`
	PetBreed   string `json:"pet_breed"`
	OwnerName  string `json:"owner_name"`
	VetName    string `json:"vet_name"`
}

// prescriptionTemplate renders a self-contained, print-ready A4 document. It is
// parsed once at package load. Using html/template guarantees every field is
// contextually escaped, so prescription content cannot inject markup.
var prescriptionTemplate = template.Must(template.New("prescription").
	Funcs(template.FuncMap{
		"fmtDate": formatBRDate,
		"add1":    func(i int) int { return i + 1 },
	}).
	Parse(prescriptionHTML))

// renderPrescriptionHTML writes the printable HTML document for doc to w.
func renderPrescriptionHTML(w io.Writer, doc *PrintablePrescription) error {
	return prescriptionTemplate.Execute(w, doc)
}

// formatBRDate formats a prescription date as DD/MM/YYYY, matching the pt-BR
// convention used across the product. A zero date renders empty.
func formatBRDate(doc *PrintablePrescription) string {
	if doc == nil || doc.Prescription == nil || doc.Date.IsZero() {
		return ""
	}
	return doc.Date.Format("02/01/2006")
}

// prescriptionHTML is the print layout. The body opens the browser print dialog
// on load so visiting the endpoint in a new tab prints immediately, while the
// @media print rules keep the on-screen and printed output consistent.
const prescriptionHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<title>Receituário Veterinário{{if .PetName}} - {{.PetName}}{{end}}</title>
<style>
  * { box-sizing: border-box; }
  body { font-family: Arial, Helvetica, sans-serif; color: #212B36; margin: 0; padding: 40px; }
  .sheet { max-width: 760px; margin: 0 auto; }
  h1 { font-size: 22px; margin: 0 0 4px; }
  h2 { font-size: 15px; margin: 24px 0 10px; text-transform: uppercase; letter-spacing: .5px; color: #454F5B; }
  .muted { color: #637381; font-size: 13px; }
  .divider { border: none; border-top: 1px solid #DFE3E8; margin: 16px 0; }
  .meta p { margin: 2px 0; font-size: 14px; }
  .meta strong { display: inline-block; min-width: 80px; }
  .item { border: 1px solid #DFE3E8; border-radius: 8px; padding: 12px 16px; margin-bottom: 12px; page-break-inside: avoid; }
  .item .name { font-size: 15px; font-weight: 700; }
  .item .row { font-size: 13px; color: #637381; margin-top: 4px; }
  .item .instructions { font-size: 13px; margin-top: 6px; }
  .notes { font-size: 14px; white-space: pre-wrap; }
  .signature { margin-top: 72px; text-align: center; }
  .signature .line { border-top: 1px solid #212B36; width: 280px; margin: 0 auto 6px; }
  .signature .name { font-size: 14px; }
  @media print {
    body { padding: 0; }
    .item { border-color: #000; }
  }
</style>
</head>
<body onload="window.print()">
  <div class="sheet">
    <header>
      <h1>Receituário Veterinário</h1>
      {{with fmtDate .}}<p class="muted">Emitido em {{.}}</p>{{end}}
    </header>

    <hr class="divider">

    <section class="meta">
      {{if .PetName}}<p><strong>Paciente:</strong> {{.PetName}}{{if or .PetSpecies .PetBreed}} ({{.PetSpecies}}{{if and .PetSpecies .PetBreed}} - {{end}}{{.PetBreed}}){{end}}</p>{{end}}
      {{if .OwnerName}}<p><strong>Tutor:</strong> {{.OwnerName}}</p>{{end}}
    </section>

    <h2>Prescrição</h2>
    {{range $i, $item := .Items}}
    <div class="item">
      <div class="name">{{add1 $i}}. {{$item.Medication}}{{if $item.Dosage}} — {{$item.Dosage}}{{end}}</div>
      {{if or $item.Route $item.Frequency $item.Duration}}
      <div class="row">
        {{- with $item.Route}}Via: {{.}}{{end}}
        {{- if and $item.Route (or $item.Frequency $item.Duration)}} · {{end}}
        {{- with $item.Frequency}}Frequência: {{.}}{{end}}
        {{- if and $item.Frequency $item.Duration}} · {{end}}
        {{- with $item.Duration}}Duração: {{.}}{{end}}
      </div>
      {{end}}
      {{if $item.Instructions}}<div class="instructions">{{$item.Instructions}}</div>{{end}}
    </div>
    {{end}}

    {{if .GeneralNotes}}
    <h2>Observações</h2>
    <p class="notes">{{.GeneralNotes}}</p>
    {{end}}

    <div class="signature">
      <div class="line"></div>
      <div class="name">{{if .VetName}}{{.VetName}}{{else}}Médico(a) Veterinário(a){{end}}</div>
      <div class="muted">Médico(a) Veterinário(a)</div>
    </div>
  </div>
</body>
</html>`
