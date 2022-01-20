package main

import (
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func Test_typeConv(t *testing.T) {
	if string(rune(323)) != "Ń" {
		t.Fatal("invalid type conversion")
	}
	if strconv.Itoa(323) != "323" {
		t.Fatal("invalid type conversion")
	}
}

func Test_regexPatterns(t *testing.T) {
	for _, tt := range []struct {
		sampleContent string
		want          string
		regexPattern  string
	}{
		// applicationId regex
		{`applicationId "com.mycompany.myapp"`, `"com.mycompany.myapp"`, applicationIDRegexPattern},
		{`applicationId "com.mycompany.myapp"//close comment`, `"com.mycompany.myapp"`, applicationIDRegexPattern},
		{`applicationId "com.mycompany.myapp" // far comment`, `"com.mycompany.myapp"`, applicationIDRegexPattern},
		{`applicationId 'com.mycompany.myapp'`, `'com.mycompany.myapp'`, applicationIDRegexPattern},
		{`applicationId 'com.mycompany.myapp'//close comment`, `'com.mycompany.myapp'`, applicationIDRegexPattern},
		{`applicationId 'com.mycompany.myapp' // far comment`, `'com.mycompany.myapp'`, applicationIDRegexPattern},
		{`applicationId = 'com.mycompany.myapp' // far comment`, `'com.mycompany.myapp'`, applicationIDRegexPattern},
	} {
		t.Run(tt.sampleContent, func(t *testing.T) {
			got := regexp.MustCompile(tt.regexPattern).FindStringSubmatch(tt.sampleContent)
			if len(got) == 0 {
				t.Errorf("regex(%s) didn't match for content: %s\n\n got: %s", tt.regexPattern, tt.sampleContent, got)
				return
			}
			if got[1] != tt.want {
				t.Errorf("got: (%v), want: (%v)", got[1], tt.want)
			}
		})
	}
}

func TestBuildGradleApplicationIDUpdater_UpdateApplicationID(t *testing.T) {
	tests := []struct {
		name              string
		buildGradleReader io.Reader
		newApplicationID  string

		want    UpdateResult
		wantErr bool
	}{
		// applicationId update
		{
			name:              "Updates applicationId value with single quote",
			buildGradleReader: strings.NewReader(`applicationId "com.mycompany.myappId"`),
			newApplicationID:  `"com.mynewcompany.mynewappId"`,
			want:              UpdateResult{NewContent: `applicationId "com.mynewcompany.mynewappId"`, FinalApplicationID: `"com.mynewcompany.mynewappId"`, UpdatedApplicationID: 1},
		},
		{
			name:              "Updates applicationId value with double quote",
			buildGradleReader: strings.NewReader(`applicationId 'com.mycompany.myappId'`),
			newApplicationID:  `"com.mynewcompany.mynewappId"`,
			want:              UpdateResult{NewContent: `applicationId "com.mynewcompany.mynewappId"`, FinalApplicationID: `"com.mynewcompany.mynewappId"`, UpdatedApplicationID: 1},
		},
		{
			name:              "Updates applicationId variable",
			buildGradleReader: strings.NewReader("applicationId rootProject.ext.applicationId"),
			newApplicationID:  `"com.mynewcompany.mynewappId"`,
			want:              UpdateResult{NewContent: `applicationId "com.mynewcompany.mynewappId"`, FinalApplicationID: `"com.mynewcompany.mynewappId"`, UpdatedApplicationID: 1},
		},
		{
			name:              "Adds quotation mark to newApplicationID if missing",
			buildGradleReader: strings.NewReader("applicationId rootProject.ext.applicationId"),
			newApplicationID:  `com.mynewcompany.mynewappId`,
			want:              UpdateResult{NewContent: `applicationId "com.mynewcompany.mynewappId"`, FinalApplicationID: `"com.mynewcompany.mynewappId"`, UpdatedApplicationID: 1},
		},
		{
			name:              "Adds quotation mark to newApplicationID if leading is missing",
			buildGradleReader: strings.NewReader("applicationId rootProject.ext.applicationId"),
			newApplicationID:  `com.mynewcompany.mynewappId"`,
			want:              UpdateResult{NewContent: `applicationId "com.mynewcompany.mynewappId"`, FinalApplicationID: `"com.mynewcompany.mynewappId"`, UpdatedApplicationID: 1},
		},
		{
			name:              "Adds quotation mark to newApplicationID if traling is missing",
			buildGradleReader: strings.NewReader("applicationId rootProject.ext.applicationId"),
			newApplicationID:  `"com.mynewcompany.mynewappId`,
			want:              UpdateResult{NewContent: `applicationId "com.mynewcompany.mynewappId"`, FinalApplicationID: `"com.mynewcompany.mynewappId"`, UpdatedApplicationID: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewBuildGradleApplicationIDUpdater(tt.buildGradleReader)
			got, err := u.UpdateApplicationID(tt.newApplicationID)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildGradleApplicationIDUpdater.UpdateApplicationID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BuildGradleApplicationIDUpdater.UpdateApplicationID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_removeQuotationMarks(t *testing.T) {
	tests := []struct {
		name string
		args string
		want string
	}{
		{
			name: "Nothing to remove",
			args: `FooBar`,
			want: `FooBar`,
		},
		{
			name: "Single Quote",
			args: `'FooBar'`,
			want: `FooBar`,
		},
		{
			name: "Double Quote",
			args: `"FooBar"`,
			want: `FooBar`,
		},
		{
			name: "Stress",
			args: `"'"FooBar'"''`,
			want: `FooBar`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := removeQuotationMarks(tt.args); got != tt.want {
				t.Errorf("removeQuotationMarks() = %v, want %v", got, tt.want)
			}
		})
	}
}
