package salutespeech

import (
	"bufio"
	"fmt"
	"os"
	"testing"

	"github.com/nk87rus/transcriptor/internal/model"
	"github.com/nk87rus/transcriptor/internal/service/sbertoken"
)

const testAuthKey = "SECRET"

func TestRecognize(t *testing.T) {
	af := model.AudioFile{
		// LocalFilePath: "../../../test_data/voice_2013458933.oga",
		// LocalFilePath: "../../../test_data/questions_about_go2.ogg",
		// Encoding: "OPUS",
		// MIME:     "audio/ogg",

		LocalFilePath: "../../../test_data/123.mp3",
		Encoding:      "MP3",
		MIME:          "audio/mpeg",
	}

	s := SaluteSpeechClient{token: sbertoken.NewTokenManager(testAuthKey, scope)}
	for resultDataFile, resultError := range s.Recognize(t.Context(), &af) {
		if resultError != nil {
			t.Fatal(resultError)
		}

		println(resultDataFile)
	}
}

const (
	reqID      = "5f37665e-083b-497c-b21b-b466b2a54928"
	fileID     = "792ff98a-9182-46a7-bafe-17aa0471e503"
	taskID     = "f127c3e3a4b0f9d3cbc4d0d65eff15a6"
	respFileID = "7b4ec244-1f56-406f-9daf-3ec713ff2a1b"
)

func TestCreateTask(t *testing.T) {
	s := SaluteSpeechClient{token: sbertoken.NewTokenManager(testAuthKey, scope)}
	resultData, resultError := s.createTask(t.Context(), reqID, fileID, "OPUS")
	if resultError != nil {
		t.Fatal(resultError)
	}

	println("taskID:", resultData)

}

func TestPolltask(t *testing.T) {
	s := SaluteSpeechClient{token: sbertoken.NewTokenManager(testAuthKey, scope)}
	resultData, resultError := s.pollTask(t.Context(), taskID)

	if resultError != nil {
		t.Fatal(resultError)
	}

	fmt.Printf("taskResult: %+v\n", resultData)
}

func TestFetchResult(t *testing.T) {
	s := SaluteSpeechClient{token: sbertoken.NewTokenManager(testAuthKey, scope)}
	for resultData, resultError := range s.fetchResult(t.Context(), respFileID) {
		if resultError != nil {
			t.Fatal(resultError)
		}

		fmt.Printf("%+v\n", resultData)
	}
}

func TestParseResults(t *testing.T) {
	f, err := os.OpenFile("../../../test_data/questions_about_go1.ogg.json", os.O_RDONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	fr := bufio.NewReader(f)
	for resultData, resultError := range parseResults(fr, func() error { return nil }) {
		if resultError != nil {
			t.Fatal(resultData)
		}

		fmt.Printf("res: %+v\n", resultData)
	}
}
