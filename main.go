package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "save-analyzer",
		Usage: "analyze your saves",
		Flags: []cli.Flag{
			&cli.PathFlag{
				Name:        "dir",
				Aliases:     []string{"d"},
				Usage:       "saves dir",
				DefaultText: "current dir",
			},
			&cli.DurationFlag{
				Name:    "delay",
				Aliases: []string{"sd"},
				Value:   time.Minute * 30,
				Usage:   "max delay between two saves in one game session",
			},
			&cli.IntFlag{
				Name:    "short",
				Aliases: []string{"s"},
				Value:   2,
				Usage:   "skip sessions with amount of saves less then NUMBER",
			},
			&cli.BoolFlag{
				Name:    "table",
				Aliases: []string{"t"},
				Value:   false,
				Usage:   "print sessions table",
			},
		},
		Action: func(c *cli.Context) (err error) {
			var sDir string
			if !c.IsSet("dir") {
				sDir, err = os.Executable()
				sDir = filepath.Dir(sDir)
				if err != nil {
					log.Fatalf("can't get dir error: %v", err)
				}
			} else {
				sDir = c.Path("dir")
				fInfo, err := os.Stat(sDir)
				if err != nil || !fInfo.IsDir() {
					log.Fatalf("dir path is incorrect or dir does not exist error: %v", err)
				}
			}
			saves, err := ioutil.ReadDir(sDir)
			if err != nil {
				log.Fatalf("read save dir error: %v", err)
			}
			saveDates := make([]int64, len(saves), len(saves))
			for i, save := range saves {
				if save.IsDir() {
					continue
				}
				saveDates[i] = save.ModTime().Unix()
			}
			if len(saveDates) < 1 {
				log.Fatal("saves dir is empty error")
			}

			sort.Slice(saveDates, func(i, j int) bool {
				return saveDates[i] < saveDates[j]
			})

			// total stat
			sessionsCount := 1
			var totalTime int64
			var longestSession int64

			// session stat
			var sessionTime int64
			sessionStartStamp := saveDates[0]
			sessionStopStamp := saveDates[0]
			sessionSavesCount := 1
			saveDates = saveDates[1:]

			// load settings
			oneSessionSaveDelay := int64(c.Duration("delay").Seconds())
			sessionSavesToSave := c.Int("short")
			printTable := c.Bool("table")

			var table *tablewriter.Table
			if printTable {
				table = tablewriter.NewWriter(os.Stdout)
				table.SetHeader([]string{"#", "duration", "saves", "saves rate", "session start", "session stop"})
			}

			saveSession := func(incSessCounter bool) {
				if sessionSavesCount < sessionSavesToSave {
					return
				}
				totalTime += sessionTime
				if sessionTime > longestSession {
					longestSession = sessionTime
				}
				if printTable {
					table.Append([]string{strconv.Itoa(sessionsCount), (time.Second * time.Duration(sessionTime)).String(),
						strconv.Itoa(sessionSavesCount), (time.Second * time.Duration(sessionTime/int64(sessionSavesCount))).String(),
						time.Unix(sessionStartStamp, 0).Format(time.Stamp), time.Unix(sessionStopStamp, 0).Format(time.Stamp)})
				}
				if incSessCounter {
					sessionsCount++
				}
			}

			for i, saveDate := range saveDates {
				if sdelay := saveDate - sessionStopStamp; sdelay < oneSessionSaveDelay {
					sessionTime += sdelay
					sessionSavesCount++
				} else {
					saveSession(true)
					sessionStartStamp = saveDate
					sessionTime = 0
					sessionSavesCount = 1
				}
				sessionStopStamp = saveDate
				if i == len(saveDates)-1 {
					saveSession(false)
				}
			}

			if printTable {
				table.Render()
			}
			fmt.Printf("saves: %d;\nsessions: %d;\ntotal time: %s;\nlongest session time: %s", len(saveDates)+1,
				sessionsCount, (time.Second * time.Duration(totalTime)).String(), (time.Second * time.Duration(longestSession)).String())
			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
