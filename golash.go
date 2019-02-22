package main

import (
	"os"
	"fmt"
	"strings"
	"os/exec"
	"github.com/ZadenRB/readline"
	"io"
	"regexp"
	"path/filepath"
	"strconv"
	"github.com/pborman/getopt"
	"golang.org/x/crypto/ssh/terminal"
	"bufio"
	"os/signal"
	"syscall"
	"github.com/ZadenRB/go-lexer"
)

// Matches single and double quotes: (?:"[^"\\]*(\\.[^"\\]*)*"|'[^'\\]*(\\.[^'\\]*)*')

var operatorMatcher, _ = regexp.Compile(`&{1,2}|\|&|\|{1,2}|;{1,2}|!`)

var wd, _ = os.Getwd()

var lastDir = wd

var dirChanged = true

var homeDir = os.Getenv("HOME")

var aliases = make(map[string]string)

var variables = make(map[string]string)

var pid = strconv.Itoa(os.Getpid())


var aOpt, bOpt, cOpt, COpt, eOpt, fOpt, hOpt, iOpt, mOpt, nOpt, uOpt, vOpt, xOpt bool

func execInput(input string) error {
	input = strings.TrimSuffix(input, "\n")

	if strings.HasPrefix(input, "#") {
		return nil
	}


	l := lexer.New(input, lexDelimitation)

	l.RunLexer()

	tokenChannel := l.Tokens

	for {
		tok := <-tokenChannel
		if tok.Type == ErrorToken {
			break
		} else {
			fmt.Print(tok.Type)
			fmt.Print(": ")
			fmt.Print(tok.Value)
			fmt.Println()
		}
	}

	commands := SplitLastSubmatch(input, operatorMatcher)

	for _, command := range commands {

		args := strings.Fields(command)

		if len(args) == 0 {
			continue
		}

		// Expand aliases
		args = processAliases(args)

		// Expand variables
		args = processVariables(args)

		// Remove empty arguments
		args = removeEmptyArgs(args)

		//Replace ~ with home directory in arguments if not in quotes
		/*for _, arg := range args {
			ReplaceAllStringLastSubmatch(arg, homeDirMatcher, homeDir)
		}*/

		switch args[0] {
			case "":
				return nil
			case "cd":
				if len(args) < 2 {
					err := toHomeDir()
					if err != nil {
						return err
					}
				} else {
					directory := args[1]
					isRelative := filepath.IsAbs(directory)
					if isRelative {
						directory = filepath.Join(wd, directory)
					}
					err := os.Chdir(directory)
					if err != nil {
						return err
					}
				}
				dirChanged = true
				return nil
			case "exit":
				os.Exit(0)
			}

			args = removeEmptyArgs(args)

			cmd := exec.Command(args[0], args[1:]...)

			cmd.Stderr = os.Stderr
			cmd.Stdout = os.Stdout

			err := cmd.Run()
			if err != nil {
				return err
			}
			return nil
		}
	return nil
}

func main() {

	readConfig()

	wd, _ = os.Getwd()

	aliases["e"] = "echo $1"

	variables["STR"] = "hello world!"

	os.Setenv("PATH", os.Getenv("PATH") + ":" + homeDir + "/.rvm/bin")

	prompt := filepath.Base(wd) + " ❯ "

	// Main shell options
	cFlag := getopt.Bool('c', "")
	iFlag := getopt.Bool('i', "")
	sFlag := getopt.Bool('s', "")

	// Other shell options
	/*aFlag := getopt.Bool('a', "")
	bFlag := getopt.Bool('b', "")
	CFlag := getopt.Bool('C', "")
	eFlag := getopt.Bool('e', "")
	fFlag := getopt.Bool('f', "")
	hFlag := getopt.Bool('h', "")
	iFlag := getopt.Bool('i', "")
	nFlag := getopt.Bool('n', "")
	uFlag := getopt.Bool('u', "")
	vFlag := getopt.Bool('v', "")
	xFlag := getopt.Bool('x', "")*/
	getopt.Bool('m', "")

	getopt.Parse()

	if *cFlag == true {
		*iFlag = false
		commandString := getopt.Arg(0)
		for idx, arg := range getopt.Args() {
			if idx > 0 {
				variables[strconv.Itoa(idx - 1)] = arg
			}
		}
		err := execInput(commandString)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		return
	} else if *sFlag == true || len(getopt.Args()) == 0 {
		*sFlag = true
		if terminal.IsTerminal(int(os.Stdout.Fd())) && terminal.IsTerminal(int(os.Stderr.Fd())) {
			*iFlag = true
		}
	} else {
		file, err := os.Open(getopt.Arg(0))
		if err != nil {
			fmt.Println(err)
		}
		reader := bufio.NewReader(file)
		for {
			input, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					fmt.Println(err)
				} else {
					err = execInput(input)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
					}
				}
				return
			}
			err = execInput(input)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		}
		return
	}

	// Catch signals
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Signal(syscall.SIGINT))
		signal.Notify(ch, os.Signal(syscall.SIGTERM))
		signal.Notify(ch, os.Signal(syscall.SIGTTIN))
		signal.Notify(ch, os.Signal(syscall.SIGTTOU))
		signal.Notify(ch, os.Signal(syscall.SIGTSTP))
		for {
			<-ch
		}
	}()

	r, err := readline.NewEx(&readline.Config {
		Prompt:            prompt,
		InterruptPrompt:   " ",
		HistoryFile:       "~/.goshellhist",
		HistorySearchFold: true,
	})
	if err != nil {
		panic(err)
	}
	defer r.Close()

	if *iFlag {
		for {
			wd, _ = os.Getwd()
			if _, err := os.Stat(wd); os.IsNotExist(err) {
				fmt.Println("The current working directory no longer exists, moving to parent directory")
				toParentDir(filepath.Dir(wd))
				dirChanged = true
				continue
			}
			if wd != lastDir {
				os.Setenv("OLDPWD", lastDir)
				if !dirChanged {
					fmt.Println("The current working directory seems to have changed unexpectedly, returning to home directory")
					toHomeDir()
					dirChanged = true
					continue
				} else {
					r.SetPrompt(filepath.Base(wd) + " ❯ ")
				}
			}

			input, err := r.Readline()

			if err == readline.ErrInterrupt {
				continue
			} else if err == io.EOF {
				break
			}

			dirChanged = false

			err = execInput(input)

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			lastDir = wd
		}
	}
}
