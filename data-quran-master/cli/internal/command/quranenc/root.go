package quranenc

import (
	"context"
	"data-quran-cli/internal/dl"
	"data-quran-cli/internal/util"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

func Command() *cli.Command {
	return &cli.Command{
		Name:   "quranenc",
		Action: cliAction,
		Usage:  "download data from QuranEnc.com",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "dst",
				Aliases: []string{"d"},
				Usage:   "destination directory",
				Value:   ".",
			},
			&cli.BoolFlag{
				Name:    "clear-cache",
				Aliases: []string{"cc"},
				Usage:   "clear download cache",
			},
		},
	}
}

func cliAction(c *cli.Context) error {
	// Prepare cache dir
	dstDir := c.String("dst")
	cacheDir := filepath.Join(dstDir, ".cache", "quranenc")
	if c.Bool("clear-cache") {
		os.RemoveAll(cacheDir)
	}
	os.MkdirAll(cacheDir, os.ModePerm)

	// Download index page from Quranenc
	err := downloadIndexPage(cacheDir)
	if err != nil {
		return err
	}

	// Parse URLs from the index page
	downloadRequests, err := parseIndexPage(cacheDir)
	if err != nil {
		return err
	}

	// Filter download request that not cached
	var requests []dl.Request
	for _, r := range downloadRequests {
		dstPath := filepath.Join(cacheDir, r.FileName)
		if !util.FileExist(dstPath) {
			requests = append(requests, r)
		}
	}

	// Batch download the request
	ctx := context.Background()
	err = dl.BatchDownload(ctx, cacheDir, requests, nil)
	if err != nil {
		return err
	}

	// Parse all XML
	dataList, err := parse(cacheDir)
	if err != nil {
		return err
	}

	// Clean data
	for i, data := range dataList {
		if fnCleaner, exist := cleanerList[data.Meta.ID]; exist {
			logrus.Printf("cleaning %s", data.Meta.ID)
			dataList[i] = fnCleaner(data)
		}
	}

	// Write to file
	err = cleanDstDir(dstDir)
	if err != nil {
		return err
	}

	err = write(dstDir, dataList)
	if err != nil {
		return err
	}

	return nil
}
