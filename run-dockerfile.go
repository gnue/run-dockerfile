package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

// コマンドの使い方
func usage() {
	cmd := os.Args[0]
	fmt.Fprintf(os.Stderr, "usage: %s [options] [file...]\n", filepath.Base(cmd))
	flag.PrintDefaults()
	os.Exit(0)
}

// Dockerfile の実行
func runDockerfile(path string) error {
	var file *os.File
	var err error

	if len(path) == 0 {
		file = os.Stdin
	} else {
		file, err = os.Open(path)
		if err != nil {
			// エラー処理をする
			return err
		}
		defer file.Close()
	}

	scanner := bufio.NewScanner(file)

	// 正規表現のコンパイル
	re := regexp.MustCompile("(?i)^ *(ONBUILD +)?([A-Z]+) +([^#]+)")
	lineno := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineno += 1
		match := re.FindStringSubmatch(line)
		if match != nil {
			//onbuild := match[1]
			instruction := match[2]
			args := match[3]

			switch instruction {
			case "FROM":
				// 何もしない
			case "MAINTAINER":
				// 何もしない
			case "RUN":
				// スクリプトを実行
				cmd := exec.Command("/bin/sh", "-c", args)
				out, err := cmd.Output()
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s:%d: %s : %s\n", path, lineno, err, args)
					return err
				}
				fmt.Printf("%s", out)
			case "CMD":
			case "EXPOSE":
				// iptables があれば書換える
			case "ENV":
				// 環境変数を設定
			case "ADD":
				// ファイルを追加
			case "ENTRYPOINT":
			case "VOLUME":
				// 何もしない
			case "USER":
				// ユーザを設定
			case "WORKDIR":
				// 作業ディレクトリを変更
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	return err
}

func main() {
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help {
		usage()
	}

	if len(flag.Args()) == 0 {
		runDockerfile("Dockerfile")
	} else {
		for _, v := range flag.Args() {
			runDockerfile(v)
		}
	}
}
