package squirt_test

import (
	"testing"

	"github.com/sanity-io/go-squirt/squirt"
)

type BananStruct struct {
	Alacazam int
	Foosh    []string
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
