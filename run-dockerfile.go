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

// オプション
type options struct {
	host string // リモートホスト
}

// 実行コンテキスト
type context struct {
	path   string // 処理中のファイル
	lineno uint   // 処理中の行番号
}

// コマンドの使い方
func usage() {
	cmd := os.Args[0]
	fmt.Fprintf(os.Stderr, "usage: %s [options] [file...]\n", filepath.Base(cmd))
	flag.PrintDefaults()
	os.Exit(0)
}

// コマンドの実行
func (ctx *context) execl(name string, arg ...string) (*exec.Cmd, error) {
	cmd := exec.Command(name, arg...)
	out, err := cmd.Output()
	if 0 < len(out) {
		fmt.Printf("%s", out)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s:%d: %s : %s\n", ctx.path, ctx.lineno, err, arg)
	}

	return cmd, err
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
	ctx := context{path: path, lineno: 0}

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
		ctx.lineno += 1
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
				_, err := ctx.execl(command[0], append(command[1:], args)...)
				if err != nil {
					return err
				}
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

					_, err := ctx.execl(copy[0], src, dst)
					if err != nil {
						return err
					}
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
