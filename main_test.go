package main

import "testing"

func TestEmail(t *testing.T) {

	err := sendEmail()
	if err != nil {
		t.Errorf("want: %v, got: %v", nil, err)
	}
	
}