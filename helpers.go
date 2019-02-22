package main

import (
	"os"
	"path/filepath"
	"fmt"
	"bufio"
	"strings"
	"strconv"
	"regexp"
	"path"
)

var configPath = filepath.Join(homeDir, ".goshellrc")

var savedVariableMatcher, _ = regexp.Compile(`\$[A-Za-z]+`)

func readConfig() {

	config, err := os.Open(filepath.Join(homeDir, ".goshellrc"))

	if err != nil {
		fmt.Println(err)
	}

	defer config.Close()

	scanner := bufio.NewScanner(config)
	scanner.Split(bufio.ScanLines)
	i := 1
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), "#")[0]
		command := strings.Split(line, " ")[0]
		if len(strings.TrimSpace(line)) > 0 {
			err := execInput(line)
			if err != nil {
				fmt.Fprint(os.Stderr, configPath + ":" + strconv.Itoa(i) + ": ")
				if err.Error() == "exec: \"" + command + "\": executable file not found in $PATH" {
					fmt.Fprint(os.Stderr, "command not found: " + command)
				}
				fmt.Fprint(os.Stderr, "\n")
			}
		}
		i++
	}
}

func removeEmptyArgs(args []string) []string {

	deleted := 0
	for i := range args {
		j := i - deleted
		if len(strings.TrimSpace(args[j])) == 0 {
			args = args[:j+copy(args[j:], args[j+1:])]
			deleted++
		}
	}

	return args
}

func processAliases(args []string) []string {
	if val, ok := aliases[args[0]]; ok {
		args[0] = val
		args = strings.Fields(strings.Join(args, " "))
	}
	return args
}

func processVariables(args []string) []string {
	/*for idx, arg := range args {
		indices := doubleQuoteStringMatcher.FindAllStringIndex(arg, -1)
		args[idx] = ReplaceAllStringLastSubmatchFunc(arg, savedVariableMatcher, func(s string) string {
			if inRanges(indices, strings.Index(arg, s)) {
				varName := strings.Split(s, "$")[1]
				return variables[varName]
			} else {
				return s
			}
		})
	}*/
	return args
}

func inRanges(ranges [][]int, n int) bool {
	for _, r := range ranges {
		if n >= r[0] && n < r[1] {
			return true
		}
	}
	return false
}

func replaceAtIndex(in string, new string, i int) string {
	out := in[:i] + new + in[i+1:]
	return out
}

func toHomeDir() error {
	err := os.Chdir(homeDir)
	if err != nil {
		return err
	} else {
		return nil
	}
}

func toParentDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		toParentDir(path.Dir(dir))
	} else {
		os.Chdir(dir)
	}
}

func FindAllStringLastSubmatch(s string, matcher *regexp.Regexp) []string {
	res := make([]string, 0)
	matches := matcher.FindAllStringSubmatchIndex(s, -1)
	for _, match := range matches {
		relevantIndices := match[len(match) - 2:]
		res = append(res, s[relevantIndices[0]:relevantIndices[1]])
	}
	return res
}

func ReplaceAllStringLastSubmatch(s string, matcher *regexp.Regexp, new string) string {
	matches := matcher.FindAllStringSubmatchIndex(s, -1)
	for _, match := range matches {
		relevantIndices := match[len(match) - 2:]
		if relevantIndices[0] == -1 || relevantIndices[1] == -1 {
			continue
		}
		replace := s[relevantIndices[0]:relevantIndices[1]]
		s = strings.Replace(s, replace, new, -1)
	}
	return s
}

func ReplaceAllStringLastSubmatchFunc(s string, matcher *regexp.Regexp, fn func(string) string) string {
	matches := matcher.FindAllStringSubmatchIndex(s, -1)
	for _, match := range matches {
		relevantIndices := match[len(match) - 2:]
		if relevantIndices[0] == -1 || relevantIndices[1] == -1 {
			continue
		}
		replace := s[relevantIndices[0]:relevantIndices[1]]
		replacement := fn(replace)
		s = strings.Replace(s, replace, replacement, -1)
	}
	return s
}

func SplitLastSubmatch(s string, matcher *regexp.Regexp) []string {
	matches := matcher.FindAllStringSubmatchIndex(s, -1)
	res := make([]string, 0)
	for _, match := range matches {
		relevantIndices := match[len(match) - 2:]
		if relevantIndices[0] == -1 || relevantIndices[1] == -1 {
			continue
		}
		res = append(res, s[:relevantIndices[0]])
		res = append(res, s[relevantIndices[0]:relevantIndices[1]])
		res = append(res, s[relevantIndices[1]:])
	}
	if len(res) == 0 {
		res = append(res, s)
	}
	return res
}