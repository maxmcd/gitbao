package config

// The config package parses configuration files
// for bao creation

import (
	"regexp"
	"strconv"
	"strings"
)

type Config struct {
	Port   int64
	EnvVar map[string]string
}

const (
	portRegexString    = "^\\s*PORT\\s*(\\d*).*$"
	envVarRegexString  = "^\\s*(\\S*?)\\s*=\\s*(\\S*).*$"
	commentRegexString = "^\\s*#.*$"
)

func Parse(config string) (response Config, err error) {

	response.EnvVar = make(map[string]string)

	portRegex := regexp.MustCompile(portRegexString)
	envVarRegex := regexp.MustCompile(envVarRegexString)

	lines := strings.Split(config, "\n")
	for _, line := range lines {
		var comment bool
		comment, err = regexp.MatchString(commentRegexString, line)
		if err != nil {
			return
		}
		if comment == true {
			continue
		}

		var port bool
		port, err = regexp.MatchString(portRegexString, line)
		if err != nil {
			return
		}
		if port == true {
			matches := portRegex.FindStringSubmatch(line)
			var portInt int
			portInt, err = strconv.Atoi(matches[1])
			if err != nil {
				return
			}
			response.Port = int64(portInt)
		}

		var envVar bool
		envVar, err = regexp.MatchString(envVarRegexString, line)
		if err != nil {
			return
		}
		if envVar == true {
			matches := envVarRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				response.EnvVar[matches[1]] = matches[2]
			}
		}
	}
	return
}
