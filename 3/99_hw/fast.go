package main

import (
	"fmt"
	"github.com/mailru/easyjson"
	"hw3/user"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

func FastSearch(out io.Writer) {
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}

	fileContents, err := ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	r := regexp.MustCompile("@")
	seenBrowsers := []string{}
	uniqueBrowsers := 0
	foundUsers := ""

	lines := strings.Split(string(fileContents), "\n")
	var users []user.User

	for i, line := range lines {
		user := user.User{}
		err := easyjson.Unmarshal([]byte(line), &user)
		if err != nil {
			panic(err)
		}
		users = append(users, user)

		isAndroid := false
		isMSIE := false

		for _, browser := range user.Browser {
			err := easyjson.Unmarshal([]byte(line), &user)
			if err != nil {
				continue
			}
			if strings.Contains(browser, "Android") {
				isAndroid = true
				notSeenBefore := true
				for _, i := range seenBrowsers {
					if i == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			} else if strings.Contains(browser, "MSIE") {
				isMSIE = true
				notSeenBefore := true
				for _, i := range seenBrowsers {
					if i == browser {
						notSeenBefore = false
					}
				}
				if notSeenBefore {
					seenBrowsers = append(seenBrowsers, browser)
					uniqueBrowsers++
				}
			}
		}
		if isMSIE && isAndroid {
			email := r.ReplaceAllString(user.Email, " [at] ")
			foundUsers += fmt.Sprintf("[%d] %s <%s>\n", i, user.Name, email)
		}
	}

	fmt.Fprintln(out, "found users:\n"+foundUsers)
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsers))

}
