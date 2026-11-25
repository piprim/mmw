package enum

import (
	"errors"
	"testing"
)

func TestCleanName(t *testing.T) {
	tests := []struct {
		label   string
		want    string
		wantErr error
	}{
		{label: "RDV d'estimation", want: "rdv_d_estimation_ID"},
		{label: "Événement d’entreprise", want: "evenement_d_entreprise_ID"},
		{label: "Relance R1", want: "relance_r1_ID"},
		{label: "Cœur de cible", want: "coeur_de_cible_ID"},
		{label: "ŒUVRE", want: "oeuvre_ID"},
		{label: "ÉTÉ", want: "ete_ID"},
		{label: "Ex æquo", want: "ex_aequo_ID"},
		{label: "  --Congé--  ", want: "conge_ID"},
		{label: "relance R1", want: "relance_r1_ID"},
		{label: "ccialEquipeMeeting", want: "ccial_equipe_meeting_ID"},
		{label: "agendaGoogle", want: "agenda_google_ID"},
		{label: "r1Bis", want: "r1_bis_ID"},
		{label: "relanceDdv", want: "relance_ddv_ID"},
		{label: "DDVProposition", want: "ddv_proposition_ID"},
		{label: "étéIndien", want: "ete_indien_ID"},
		{label: "RDV", want: "rdv_ID"},
		{label: "", wantErr: ErrEmptyName},
		{label: "---", wantErr: ErrEmptyName},
		{label: "1er RDV", wantErr: ErrLeadingDigit},
		{label: "2024", wantErr: ErrLeadingDigit},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			got, err := CleanName(tt.label)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CleanName(%q) error = %v, want %v", tt.label, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("CleanName(%q) = %q, want %q", tt.label, got, tt.want)
			}
		})
	}
}
