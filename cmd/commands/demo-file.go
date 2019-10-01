package commands

import (
	"log"
	"math"
	mathrand "math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	osext "github.com/kardianos/osext"

	"github.com/palantir/stacktrace"

	"github.com/damoonazarpazhooh/chunker/internal/jsonutil"
	utils "github.com/damoonazarpazhooh/chunker/pkg/utils"
	"github.com/urfave/cli"
)

var (
	defaultSize = "500"
)

// Sample ...
var Sample = cli.Command{
	Name:    "Sample",
	Aliases: []string{"sample"},
	Usage:   "generating sample files to chunk and merge",
	Subcommands: []cli.Command{
		singleSampleFile,
		multipleSampleFiles,
	},
}

// singleSampleFile ...
var singleSampleFile = cli.Command{
	Name:    "File",
	Aliases: []string{"file"},
	Usage:   "generates a sample file to chunk and merge",
	Description: `this command helps with generating random files.
	you set random file size in megabytes by using -size flag.
	if you don't set a size value , it would use a default value of 100mb for file size.
	and the argument is the path generated file is going to be stored in.
	if you don't pass in any arguments for path, sample files will be stored in [./tmp]

	`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "size",
			Value: defaultSize,
			Usage: "file size in mb",
		},
	},
	Action: func(ctx *cli.Context) error {
		path := ctx.Args().First()
		if len(path) == 0 {
			path = "tmp"
			selfPath, _ := osext.ExecutableFolder()
			path = utils.PathJoin(selfPath, path)
		}
		path, _ = filepath.Abs(path)

		os.MkdirAll(path, 0700)
		sizeString := ctx.String("size")
		if len(sizeString) == 0 {
			sizeString = defaultSize
		}
		size, err := strconv.Atoi(sizeString)
		if err != nil {
			log.Fatal(err)
		}

		size = (size * 1 << 10)
		// size = size * humanize.MiByte
		err = createRandomFile(path, size)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	},
}

// multipleSampleFiles ...
var multipleSampleFiles = cli.Command{
	Name:    "demo",
	Aliases: []string{"demo"},
	Usage:   "generates a multiple random files to chunk and merge",
	Description: `this command is used for rapid testing and has a bunch of predefined path
	for files which will be stored in the same directory as the binary in [./tmp] folder.
	you set random files size in megabytes by using -size flag.
	if you don't set a size value , it would use a default value of 100 mb for
	file sizes.`,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "size",
			Value: defaultSize,
			Usage: "file size in mb",
		},
	},
	Action: func(ctx *cli.Context) error {
		files := []string{
			"file-1",
			// "dir-1/file-2",
			// "dir-1/subdir-1/file-3",
		}
		selfPath, _ := osext.ExecutableFolder()
		path := ctx.Args().First()
		if len(path) == 0 {
			path = "tmp"
		}
		path = utils.PathJoin(selfPath, path)
		os.MkdirAll(path, 0700)
		sizeString := ctx.String("size")
		if len(sizeString) == 0 {
			sizeString = defaultSize
		}
		size, err := strconv.Atoi(sizeString)
		if err != nil {
			log.Fatal(err)
		}
		// size = size * humanize.MiByte
		size = (size * 1 << 10)

		err = createRandomFiles(path, files, size)
		if err != nil {
			log.Fatal(err)
		}
		return nil
	},
}

func createRandomFiles(rootDir string, files []string, maxSize int) error {
	for _, v := range files {
		pref := prefixes(v)
		if len(pref) > 0 {
			for _, vv := range pref {
				os.Mkdir(utils.PathJoin(rootDir, vv), 0700)
			}
		}
		err := createRandomFile(utils.PathJoin(rootDir, v), maxSize)
		if err != nil {
			err = stacktrace.Propagate(err, "could not create multiple files due to error in creating file at (%s)", utils.PathJoin(rootDir, v))
			return err
		}
	}
	return nil
}

// Gen 1 gb null
// dd if=/dev/zero of=file-1 count=1024 bs=1048576
// Gen 10 gb not null
// dd if=/dev/urandom of=file-1 bs=1048576 count=1048576

func createRandomFile(path string, maxSize int) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		err = stacktrace.Propagate(err, "Can't open %s for writing", path)
		return err
	}
	defer file.Close()
	size := maxSize/2 + mathrand.Int()%(maxSize/2)
	loremString := path + `---Lorem ipsum dolor sit amet, consectetur adipiscing elit. Proin facilisis mi sapien, vitae accumsan libero malesuada in. Suspendisse sodales finibus sagittis. Proin et augue vitae dui scelerisque imperdiet. Suspendisse et pulvinar libero. Vestibulum id porttitor augue. Vivamus lobortis lacus et libero ultricies accumsan. Donec non feugiat enim, nec tempus nunc. Mauris rutrum, diam euismod elementum ultricies, purus tellus faucibus augue, sit amet tristique diam purus eu arcu. Integer elementum urna non justo fringilla fermentum. Pellentesque habitant morbi tristique senectus et netus et malesuada fames ac turpis egestas. Quisque sollicitudin elit in metus imperdiet, et gravida tortor hendrerit. In volutpat tellus quis sapien rutrum, sit amet cursus augue ultricies. Morbi tincidunt arcu id commodo mollis. Aliquam laoreet purus sed justo pulvinar, quis porta risus lobortis. In commodo leo id porta mattis.`
	byteSizeOfDefaultLorem := len([]byte(loremString))
	repetitions := int(math.Round(float64(size / byteSizeOfDefaultLorem)))
	for i := 0; i < repetitions; i++ {
		enc, _ := jsonutil.EncodeJSONWithIndentation(map[int]string{
			i: (loremString),
		})
		file.Write([]byte(enc))
	}

	// buffer := make([]byte, 32*1024)
	// for size > 0 {
	// 	bytes := size
	// 	if bytes > cap(buffer) {
	// 		bytes = cap(buffer)
	// 	}
	// 	cryptorand.Read(buffer[:bytes])
	// 	bytes, err = file.Write(buffer[:bytes])
	// 	if err != nil {
	// 		err = stacktrace.Propagate(err, "Failed to write to %s", path)
	// 		return err
	// 	}
	// 	size -= bytes
	// }
	return nil
}

func prefixes(s string) []string {
	components := strings.Split(s, "/")
	result := []string{}
	for i := 1; i < len(components); i++ {
		result = append(result, strings.Join(components[:i], "/"))
	}
	return result
}
