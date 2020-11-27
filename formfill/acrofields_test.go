package formfill

import "testing"

func TestDAParse(t *testing.T) {
	da := "/Helv 9 Tf 0 g"

	conf, err := splitDAelements(da)
	if err != nil {
		t.Fatal(err)
	}
	if conf.font != "Helv" {
		t.Errorf("expected /Helv got %s", conf.font)
	}
	if conf.size != 9 {
		t.Errorf("expected 9 got %v", conf.size)
	}
	if conf.color != nil {
		t.Errorf("expected nil color, got %v", conf.color)
	}
}
