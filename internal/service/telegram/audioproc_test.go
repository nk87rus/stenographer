package telegram

import (
	"fmt"
	"testing"

	"github.com/nbonaparte/audiotags"
)

var filePath = "../../../test_data/voice_2013458933.oga"

func TestTags(t *testing.T) {
	props, audioProps, err := audiotags.Read(filePath)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("props: %+v\naProps: %+v\n", props, audioProps)
}
