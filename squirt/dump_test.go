package squirt_test

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/sanity-io/go-squirt/squirt"
	"github.com/stretchr/testify/assert"
)

type BananStruct struct {
	Alacazam int
	Foosh    []string
}

func performDumpTests(t *testing.T, suiteName string, cases interface{}) {
	referenceFileName := "testdata/" + suiteName + ".dump"
	dump := squirt.Sdump(cases)
	reference, err := ioutil.ReadFile(referenceFileName)
	if os.IsNotExist(err) || true {
		ioutil.WriteFile(referenceFileName, []byte(dump), 0644)
		return
	}
	assert.Equal(t, string(reference), dump)
}

func TestDump_primitives(t *testing.T) {
	performDumpTests(t, "primitives", []interface{}{
		false, true, 7, uint8(130), float32(12.3), float64(12.3),
		complex64(12 + 10.5i), complex128(-1.2 - 0.1i),
		nil, interface{}(nil),
	})
}

func TestSquirt_dumpBasicTypes(t *testing.T) {
	squirt.Dump(false)
	squirt.Dump(interface{}(true))
	squirt.Dump([]interface{}{"feh"})
	squirt.Dump(map[string]interface{}{"fnah": 7})
	squirt.Dump(7)
	squirt.Dump("feh, \"man\"")
	squirt.Dump(uint8(130))
	squirt.Dump(float32(127.3))
	squirt.Dump(float64(127.3))
	squirt.Dump(complex64(12 + 10.5i))
	squirt.Dump(complex128(-1.2 - 0.1i))
	squirt.Dump(nil)
	squirt.Dump(interface{}(nil))
	squirt.Dump([]interface{}{nil})
	squirt.Dump(
		struct {
			Abra    string
			Cadabra int
		}{
			Abra:    "abra",
			Cadabra: 7,
		},
	)
	squirt.Dump(
		BananStruct{
			Alacazam: 7,
			Foosh:    []string{"foo", "bar"},
		},
	)
	squirt.Dump(
		&BananStruct{
			Alacazam: 7,
			Foosh:    []string{"foo", "bar"},
		},
	)
	squirt.Dump(
		map[string]interface{}{
			"fnah": 7,
			"bargh": &BananStruct{
				Alacazam: 12,
				Foosh:    []string{"fneh", "heh"},
			},
			"Boing": 72.1,
			"feh":   []byte("fink"),
		},
	)

}
