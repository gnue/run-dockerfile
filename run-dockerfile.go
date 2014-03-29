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

type options struct {
	host string
}

// コマンドの使い方
func usage() {
	cmd := os.Args[0]
	fmt.Fprintf(os.Stderr, "usage: %s [options] [file...]\n", filepath.Base(cmd))
	flag.PrintDefaults()
	os.Exit(0)
}

// Dockerfile の実行
func runDockerfile(path string, opts *options) error {
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
	re_args := regexp.MustCompile("(?i)^([^ ]+) +(.+)")

	var workdir string
	lineno := 0

	command := []string{"/bin/sh", "-c"}
	copy := []string{"/bin/cp"}

	if 0 < len(opts.host) {
		ssh, err := exec.LookPath("ssh")
		if err != nil {
			return err
		}
		scp, err := exec.LookPath("scp")
		if err != nil {
			return err
		}
		command = []string{ssh, opts.host}
		copy = []string{scp}

	}

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
				if 0 < len(workdir) {
					args = fmt.Sprintf("cd %s; %s", workdir, args)
				}
				cmd := exec.Command(command[0], append(command[1:], args)...)
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
				match_args := re_args.FindStringSubmatch(args)
				if match_args != nil {
					os.Setenv(match_args[1], match_args[2])
				}
			case "ADD":
				// ファイルを追加
				match_args := re_args.FindStringSubmatch(args)
				if match_args != nil {
					src := match_args[1]
					dst := match_args[2]

					if 0 < len(workdir) && !filepath.IsAbs(dst) {
						// コピー先のパスを生成する
						dst = filepath.Join(workdir, dst)
					}

					if 0 < len(opts.host) {
						// リモート・パスを生成する
						dst = fmt.Sprintf("%s:%s", opts.host, dst)
					}

					cmd := exec.Command(copy[0], src, dst)
					out, err := cmd.Output()
					if err != nil {
						fmt.Fprintf(os.Stderr, "%s:%d: %s : %s\n", path, lineno, err, args)
						return err
					}
					fmt.Printf("%s", out)
				}
			case "ENTRYPOINT":
			case "VOLUME":
				// 何もしない
			case "USER":
				// ユーザを設定
			case "WORKDIR":
				// 作業ディレクトリを変更
				workdir = args
			}
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	return err
}

func main() {
	var opts options

	flag.StringVar(&opts.host, "H", "", "host")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	if *help {
		usage()
	}

	if len(flag.Args()) == 0 {
		runDockerfile("Dockerfile", &opts)
	} else {
		for _, v := range flag.Args() {
			runDockerfile(v, &opts)
		}
	}
}
