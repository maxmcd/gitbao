package config

import "testing"

const (
	sample1 = `
    # This is your bao config file, define 
    # a port and any env variables 
    PORT 8000
    THIS='that'
    THAT="this"`
	sample2 = `
        # This is your bao config file, define 
    # a port and any env variables 
    # PORT 8001
    #THIS=that
        PORT 8000
        THIS = th'at
        THAT="th#is"`
)

func TestParse(t *testing.T) {
	sample1Ans := Config{
		Port: 8000,
		EnvVar: map[string]string{
			"THIS": "that",
			"THAT": "this",
		},
	}
	response, err := Parse(sample1)
	if err != nil {
		panic(err)
	}
	if response.Port != sample1Ans.Port {
		t.Errorf("Wrong port number %d", response.Port)
	}
	for key, value := range response.EnvVar {
		if sample1Ans.EnvVar[key] != value {
			t.Errorf("%s and %s envvars not valid", key, value)
		}
	}

	sample2Ans := Config{
		Port: 8000,
		EnvVar: map[string]string{
			"THIS": `th\'at`,
			"THAT": "th#is",
		},
	}
	response, err = Parse(sample2)
	if err != nil {
		panic(err)
	}
	if response.Port != sample2Ans.Port {
		t.Errorf("Wrong port number")
	}
	for key, value := range response.EnvVar {
		if sample2Ans.EnvVar[key] != value {
			t.Errorf("'%s' and '%s' envvars not valid", key, value)
		}
	}
}
