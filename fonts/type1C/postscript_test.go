package type1C

import (
	"io/ioutil"
	"testing"
)

func TestParseCFF(t *testing.T) {
	for _, file := range []string{
		"test/ATUFBP+CMEX10.cff",
		"test/FRHWBU+CMR10.cff",
		"test/HUKATE+CMTI10.cff",
		"test/KBIKOH+CMMI7.cff",
		"test/KZJDOM+CMBX10.cff",
		"test/LKCSBW+CMSY10.cff",
		"test/MUPALF+CMMI10.cff",
		"test/PMSRXR+CMR12.cff",
		"test/SLMKHE+TeX-feymr10.cff",
		"test/SZNNBG+CMBX12.cff",
		"test/YPTQCA+CMR17.cff",
	} {
		b, err := ioutil.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		p := cffParser{src: b}
		_, err = p.parse()
		if err != nil {
			t.Fatal(err)
		}
	}
}
