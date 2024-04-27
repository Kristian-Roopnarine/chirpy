package main

import (
	"fmt"
	"testing"
)

type TestCase struct {
	testStr     string
	expectedStr string
}

func TestNoBadWords(t *testing.T) {
	cases := []TestCase{
		{
			testStr:     "This word contains no bad words",
			expectedStr: "This word contains no bad words",
		},
		{
			testStr:     "",
			expectedStr: "",
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			msg := cleanInput(c.testStr)
			if msg != c.expectedStr {
				t.Errorf("expected message to remain unchanged")
				return
			}
		})
	}

}

func TestBadWords(t *testing.T) {
	cases := []TestCase{
		{
			testStr:     "kerfuffle this is a test",
			expectedStr: "**** this is a test",
		},
		{
			testStr:     "sharbert this is a test",
			expectedStr: "**** this is a test",
		},
		{
			testStr:     "fornax this is a test",
			expectedStr: "**** this is a test",
		},
		{
			testStr:     "Fornax this is sharbert test",
			expectedStr: "**** this is **** test",
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			msg := cleanInput(c.testStr)
			if msg != c.expectedStr {
				fmt.Println(msg)
				t.Errorf("did not remove bad words properly")
				return
			}
		})
	}
}

func TestBadWordPunctuation(t *testing.T) {
	cases := []TestCase{
		{
			testStr:     "kerfuffle! this is a test",
			expectedStr: "kerfuffle! this is a test",
		},
		{
			testStr:     "sharbert? this is a test",
			expectedStr: "sharbert? this is a test",
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			msg := cleanInput(c.testStr)
			if msg != c.expectedStr {
				fmt.Println(msg)
				t.Errorf("expected message to remain unchanged")
				return
			}
		})
	}

}
