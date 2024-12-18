package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/logrusorgru/aurora/v4"
	"github.com/projectdiscovery/goflags"
	"github.com/rix4uni/linkinspector/banner"
)

type Options struct {
	InputTargetHost string
	InputFile       string
	Passive         bool
	MatchCode       string
	MatchLength     string
	MatchType       string
	MatchSuffix     string
	// FilterCode      string
	// FilterLength    string
	// FilterType      string
	// FilterSuffix    string
	Output       string
	AppendOutput string
	JSONOutput   bool
	JSONtype     string
	Threads      int
	UserAgent    string
	Verbose      bool
	Version      bool
	Silent       bool
	NoColor      bool
	Timeout      int
	Insecure     bool
	Delay        time.Duration
}

// Define the flags
func ParseOptions() *Options {
	options := &Options{}
	flagSet := goflags.NewFlagSet()
	flagSet.SetDescription(`linkinspector is a command-line tool that analyzes URLs to retrieve HTTP status codes, content lengths, and content types.`)

	createGroup(flagSet, "input", "Input",
		flagSet.StringVarP(&options.InputTargetHost, "target", "u", "", "Single URL to check"),
		flagSet.StringVarP(&options.InputFile, "list", "l", "", "File containing list of URLs to check"),
	)

	createGroup(flagSet, "probes", "Probes",
		flagSet.BoolVar(&options.Passive, "passive", false, "Enable passive mode to skip requests for specific extensions"),
	)

	createGroup(flagSet, "matchers", "Matchers",
		flagSet.StringVarP(&options.MatchCode, "match-code", "mc", "", "Match response with specified status code (e.g., -mc 200,302)"),
		flagSet.StringVarP(&options.MatchLength, "match-length", "ml", "", "Match response with specified content length (e.g., -ml 100,102)"),
		flagSet.StringVarP(&options.MatchType, "match-type", "mt", "", "Match response with specified content type (e.g., -mt \"application/octet-stream,text/html\")"),
		flagSet.StringVarP(&options.MatchSuffix, "match-suffix", "ms", "", "Match response with specified suffix name (e.g., -ms \"ZIP,PHP,7Z\")"),
	)

	// createGroup(flagSet, "filters", "Filters",
	// 	flagSet.StringVarP(&options.FilterCode, "filter-code", "fc", "", "Filter response with specified status code (e.g., -fc 403,401)"),
	// 	flagSet.StringVarP(&options.FilterLength, "filter-length", "fl", "", "Filter response with specified content length (e.g., -fl 23,33)"),
	// 	flagSet.StringVarP(&options.FilterType, "filter-type", "ft", "", "Filter response with specified content type (e.g., -ft \"text/html,image/jpeg\")"),
	// 	flagSet.StringVarP(&options.FilterSuffix, "filter-suffix", "fs", "", "Filter response with specified suffix name (e.g., -fs \"CSS,Plain Text,html\")"),
	// )

	createGroup(flagSet, "output", "Output",
		flagSet.StringVarP(&options.Output, "output", "o", "", "File to write output results"),
		flagSet.StringVar(&options.AppendOutput, "append-output", "", "File to append output results instead of overwriting"),
		flagSet.BoolVar(&options.JSONOutput, "json", false, "Output in JSON format"),
		flagSet.StringVar(&options.JSONtype, "json-type", "MarshalIndent", "Output in JSON type, MarshalIndent or Marshal"),
	)

	createGroup(flagSet, "rate-limit", "RATE-LIMIT",
		flagSet.IntVarP(&options.Threads, "threads", "t", 50, "Number of threads to use"),
	)

	createGroup(flagSet, "configurations", "Configurations",
		flagSet.StringVar(&options.UserAgent, "H", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36", "Custom User-Agent header for HTTP requests"),
	)

	createGroup(flagSet, "debug", "Debug",
		flagSet.BoolVar(&options.Verbose, "verbose", false, "Enable verbose output for debugging purposes"),
		flagSet.BoolVar(&options.Version, "version", false, "Print the version of the tool and exit"),
		flagSet.BoolVar(&options.Silent, "silent", false, "silent mode"),
		flagSet.BoolVarP(&options.NoColor, "no-color", "nc", false, "disable colors in cli output"),
	)

	createGroup(flagSet, "optimizations", "OPTIMIZATIONS",
		flagSet.IntVar(&options.Timeout, "timeout", 10, "HTTP request timeout duration (in seconds)"),
		flagSet.BoolVar(&options.Insecure, "insecure", false, "Disable TLS certificate verification"),
		flagSet.DurationVar(&options.Delay, "delay", -1*time.Nanosecond, "Duration between each HTTP request (e.g., 200ms, 1s)"),
	)

	_ = flagSet.Parse()

	return options
}

func createGroup(flagSet *goflags.FlagSet, groupName, description string, flags ...*goflags.FlagData) {
	flagSet.SetGroup(groupName, description)
	for _, currentFlag := range flags {
		currentFlag.Group(groupName)
	}
}

// Struct for JSON output
type JSONOutput struct {
	Host string `json:"host"`
	Type string `json:"type"`
	Data struct {
		StatusCode    int64  `json:"status_code,omitempty"`
		ContentLength int64  `json:"content_length,omitempty"`
		ContentType   string `json:"content_type,omitempty"`
		Suffix        string `json:"suffix,omitempty"`
	} `json:"data"`
}

// Function to check if a value matches any of the specified filters
func matches(value string, filter string) bool {
	if filter == "" {
		return true // No filter applied
	}
	filters := strings.Split(filter, ",")
	for _, f := range filters {
		if strings.TrimSpace(f) == value {
			return true
		}
	}
	return false
}

// Check URL information and return the required output format with custom timeout, TLS, and User-Agent settings.
func getURLInfo(url string, verbose bool, timeout time.Duration, insecure bool, userAgent string, jsonOutput bool, jsonTypeFlag string, outputFile *os.File, options *Options) {
	// Create a custom HTTP client with the specified timeout and TLS settings.
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecure,
			},
		},
	}

	// Create a new HTTP request with the custom User-Agent header.
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", url, err)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	// Perform the HTTP request.
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching %s: %v\n", url, err)
		return
	}
	defer resp.Body.Close()

	// Extract response details.
	statusCode := resp.StatusCode
	contentLength := resp.ContentLength
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	// Define a map for content types and their corresponding suffixes
	validExtension := map[string]string{
		// Image
		"image/jpeg":                "[jpg]",
		"image/png":                 "[png]",
		"image/gif":                 "[gif]",
		"image/webp":                "[webp]",
		"image/x-canon-cr2":         "[cr2]",
		"image/tiff":                "[tif]",
		"image/bmp":                 "[bmp]",
		"image/heif":                "[heif]",
		"image/vnd.ms-photo":        "[jxr]",
		"image/vnd.adobe.photoshop": "[psd]",
		"image/vnd.microsoft.icon":  "[ico]",
		"image/vnd.dwg":             "[dwg]",
		"image/avif":                "[avif]",

		// Video
		"video/mp4":        "[mp4]",
		"video/x-m4v":      "[m4v]",
		"video/x-matroska": "[mkv]",
		"video/webm":       "[webm]",
		"video/quicktime":  "[mov]",
		"video/x-msvideo":  "[avi]",
		"video/x-ms-wmv":   "[wmv]",
		"video/mpeg":       "[mpg]",
		"video/x-flv":      "[flv]",
		"video/3gpp":       "[3gp]",

		// Audio
		"audio/midi":   "[mid]",
		"audio/mpeg":   "[mp3]",
		"audio/mp4":    "[m4a]",
		"audio/ogg":    "[ogg]",
		"audio/x-flac": "[flac]",
		"audio/x-wav":  "[wav]",
		"audio/amr":    "[amr]",
		"audio/aac":    "[aac]",
		"audio/x-aiff": "[aiff]",

		// Archive
		"application/epub+zip":                          "[epub]",
		"application/zip":                               "[zip]",
		"application/x-tar":                             "[tar]",
		"application/vnd.rar":                           "[rar]",
		"application/gzip":                              "[gz]",
		"application/x-bzip2":                           "[bz2]",
		"application/x-7z-compressed":                   "[7z]",
		"application/x-xz":                              "[xz]",
		"application/zstd":                              "[zstd]",
		"application/pdf":                               "[pdf]",
		"application/vnd.microsoft.portable-executable": "[exe]",
		"application/x-shockwave-flash":                 "[swf]",
		"application/rtf":                               "[rtf]",
		"application/x-iso9660-image":                   "[iso]",
		// "application/octet-stream": "[eot]",
		"application/postscript":                "[ps]",
		"application/vnd.sqlite3":               "[sqlite]",
		"application/x-nintendo-nes-rom":        "[nes]",
		"application/x-google-chrome-extension": "[crx]",
		"application/vnd.ms-cab-compressed":     "[cab]",
		"application/vnd.debian.binary-package": "[deb]",
		"application/x-unix-archive":            "[ar]",
		"application/x-compress":                "[Z]",
		"application/x-lzip":                    "[lz]",
		"application/x-rpm":                     "[rpm]",
		"application/x-executable":              "[elf]",
		"application/dicom":                     "[dcm]",

		// Documents
		"application/msword": "[doc]",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": "[docx]",
		"application/vnd.ms-excel": "[xls]",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         "[xlsx]",
		"application/vnd.ms-powerpoint":                                             "[ppt]",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": "[pptx]",

		// Font
		"application/font-woff": "[woff]",
		// "application/font-woff": "[woff2]",
		"application/font-sfnt": "[ttf]",
		// "application/font-sfnt": "[otf]",

		// Application
		"application/wasm":            "[wasm]",
		"application/vnd.android.dex": "[dex]",
		"application/vnd.android.dey": "[dey]",

		// Others
		"application/octet-stream":  "[interesting]",
		"text/plain; charset=UTF-8": "[text]",
		"text/html; charset=UTF-8":  "[html]",
		"text/html":                 "[html]",
		"application/sql":           "[sql]",
		"application/x-php":         "[php]",
	}

	suffix, exists := validExtension[contentType]
	if !exists {
		suffix = ""
	}

	// Apply matchers to filter the response.
	if !matches(fmt.Sprintf("%d", statusCode), options.MatchCode) {
		return // Skip if status code does not match.
	}
	if !matches(fmt.Sprintf("%d", contentLength), options.MatchLength) {
		return // Skip if content length does not match.
	}
	if !matches(contentType, options.MatchType) {
		return // Skip if content type does not match.
	}
	if !matches(strings.Trim(suffix, "[]"), options.MatchSuffix) {
		return // Skip if suffix does not match.
	}

	// Handle JSON output.
	if jsonOutput {
		output := JSONOutput{
			Host: url,
			Type: "REQUEST BASED",
		}
		output.Data.StatusCode = int64(statusCode)
		output.Data.ContentLength = int64(contentLength)
		output.Data.ContentType = contentType
		output.Data.Suffix = strings.Trim(suffix, "[]") // Remove brackets.

		var jsonData []byte
		if jsonTypeFlag == "Marshal" {
			jsonData, _ = json.Marshal(output)
		} else {
			jsonData, _ = json.MarshalIndent(output, "", "  ") // Pretty print the JSON.
		}
		fmt.Println(string(jsonData))
		if outputFile != nil {
			outputFile.WriteString(string(jsonData) + "\n")
		}
		return
	}

	// Handle non-verbose and verbose output.
	outputLine := ""
	if verbose {
		if options.NoColor {
			outputLine = fmt.Sprintf("REQUEST BASED: %s [%d] [%d] [%s] %s\n", url, statusCode, contentLength, contentType, suffix)
		} else {
			outputLine = fmt.Sprintf("%s: %s [%d] [%d] [%s] %s\n", aurora.Bold(aurora.Blue("REQUEST BASED")), url, aurora.Green(statusCode), aurora.Magenta(contentLength), aurora.Magenta(contentType), aurora.Yellow(suffix))
		}
	} else {
		if options.NoColor {
			outputLine = fmt.Sprintf("%s [%d] [%d] [%s] %s\n", url, statusCode, contentLength, contentType, suffix)
		} else {
			outputLine = fmt.Sprintf("%s [%d] [%d] [%s] %s\n", url, aurora.Green(statusCode), aurora.Magenta(contentLength), aurora.Magenta(contentType), aurora.Yellow(suffix))
		}
	}
	fmt.Print(outputLine)
	if outputFile != nil {
		outputFile.WriteString(outputLine)
	}
}

// Skip requests based on file extensions when the -passive flag is true.
func processURL(url string, passive bool, verbose bool, timeout time.Duration, insecure bool, userAgent string, wg *sync.WaitGroup, sem chan struct{}, delay time.Duration, jsonOutput bool, jsonTypeFlag string, outputFile *os.File, options *Options) {
	defer wg.Done()
	// Acquire a spot in the semaphore
	sem <- struct{}{}

	defer func() {
		// Release the spot in the semaphore when done
		<-sem
	}()

	// Define a map for passive extensions and their corresponding output.
	passiveExtensions := map[string]string{
		// Image
		".jpg":  "[jpg]",
		".png":  "[png]",
		".gif":  "[gif]",
		".webp": "[webp]",
		".cr2":  "[cr2]",
		".tif":  "[tif]",
		".bmp":  "[bmp]",
		".heif": "[heif]",
		".jxr":  "[jxr]",
		".psd":  "[psd]",
		".ico":  "[ico]",
		".dwg":  "[dwg]",
		".avif": "[avif]",

		// Video
		".mp4":  "[mp4]",
		".m4v":  "[m4v]",
		".mkv":  "[mkv]",
		".webm": "[webm]",
		".mov":  "[mov]",
		".avi":  "[avi]",
		".wmv":  "[wmv]",
		".mpg":  "[mpg]",
		".flv":  "[flv]",
		".3gp":  "[3gp]",

		// Audio
		".mid":  "[mid]",
		".mp3":  "[mp3]",
		".m4a":  "[m4a]",
		".ogg":  "[ogg]",
		".flac": "[flac]",
		".wav":  "[wav]",
		".amr":  "[amr]",
		".aac":  "[aac]",
		".aiff": "[aiff]",

		// Archive
		".epub": "[epub]",
		".zip, .zip1, .zip2, .zip3, .zip4, .zip5, .zip6, .zip7, .zip8, .zip9": "[zip]",
		".tar": "[tar]",
		".rar": "[rar]",
		".gz":  "[gz]",
		".bz2": "[bz2]",
		".7z, .7z1, .7z2, .7z3, .7z4, .7z5, .7z6, .7z7, .7z8, .7z9": "[7z]",
		".xz":     "[xz]",
		".zstd":   "[zstd]",
		".pdf":    "[pdf]",
		".exe":    "[exe]",
		".swf":    "[swf]",
		".rtf":    "[rtf]",
		".iso":    "[iso]",
		".eot":    "[eot]",
		".ps":     "[ps]",
		".sqlite": "[sqlite]",
		".nes":    "[nes]",
		".crx":    "[crx]",
		".cab":    "[cab]",
		".deb":    "[deb]",
		".ar":     "[ar]",
		".Z":      "[Z]",
		".lz":     "[lz]",
		".rpm":    "[rpm]",
		".elf":    "[elf]",
		".dcm":    "[dcm]",

		// Documents
		".doc":  "[doc]",
		".docx": "[docx]",
		".xls":  "[xls]",
		".xlsx": "[xlsx]",
		".ppt":  "[ppt]",
		".pptx": "[pptx]",

		// Font
		"woff":  "[woff]",
		"woff2": "[woff2]",
		"ttf":   "[ttf]",
		"otf":   "[otf]",

		// Application
		".wasm": "[wasm]",
		".dex":  "[dex]",
		".dey":  "[dey]",

		// https://gist.github.com/ppisarczyk/43962d06686722d26d176fad46879d41
		// Programming Languages Extensions
		".vbs":         "[Visual-Basic]",
		".as":          "[ActionScript]",
		".applescript": "[AppleScript]",
		".sh, .bash, .bashrc, .ash, .zsh, .zshrc, .bats, .command, .ksh, .sh.in, .tmux, .tool": "[Shell]",
		".bat, .cmd": "[Batchfile]",
		".bib, .aux, .bbx, .cbx, .dtx, .lbx, .mkii, .mkiv, .mkvi, .toc, .tsx, .tcl, .sty, .cls": "[TeX]",
		".c":   "[C]",
		".h":   "[C/C++/Objective-C]",
		".cs":  "[C#/Smalltalk]",
		".csx": "[C#]",
		".cpp, .cc, .cp, .cxx, .c++, .C, .hxx, .h++, .inl, .ipp, .ixx, .cppm": "[C++]",
		".hh":                             "[C++/Hack]",
		".css":                            "[CSS]",
		".gocss, .go.css":                 "[CSS+GO]",
		".css.php":                        "[CSS+PHP]",
		".css.erb":                        "[CSS+Rails]",
		".cabal, .cabal.project":          "[Cabal]",
		".clj, .cljc, .edn":               "[Clojure]",
		".cljs":                           "[ClojureScript]",
		".d":                              "[D]",
		".di":                             "[D]",
		".dtd, .ent, .mod":                "[DTD]",
		".diff, .patch":                   "[Diff]",
		".erl, .hrl, .escript":            "[Erlang]",
		".gitattributes":                  "[Git Attributes]",
		".git-blame-ignore-revs":          "[Git Blame Ignore Revs]",
		".CODEOWNERS":                     "[CODEOWNERS]",
		".gitconfig":                      "[Git Config]",
		".gitignore":                      "[Git Ignore]",
		".git":                            "[Git Link]",
		".gitlog":                         "[Git Log]",
		".mailmap":                        "[Git+Mailmap]",
		".go":                             "[Go]",
		".dot, .gv":                       "[Graphviz+DOT]",
		".groovy, .gvy, .gradle":          "[GROOVY]",
		".haml":                           "[HAML]",
		".html, .htm, .shtml, .xhtml":     "[HTML]",
		".asp, .asa":                      "[HTML+ASP]",
		".yaws":                           "[HTML+Erlang]",
		".gohtml, .go.html, .tmpl":        "[HTML+GO]",
		".jsp, .jspf, .jspx, .jstl":       "[HTML+JSP]",
		".rails, .rhtml, .erb, .html.erb": "[HTML+Rails]",
		".adp":                            "[HTML+Tcl]",
		".hs, .hs-boot, .hsig":            "[Haskell]",
		".json, .jsonc":                   "[JSON]",
		".json.php":                       "[JSON+PHP]",
		".json.erb":                       "[JSON+Rails]",
		".jsx":                            "[JSX]",
		".java, .bsh":                     "[Java]",
		".properties":                     "[Java Properties]",
		".gojs":                           "[JavaScript+GO]",
		".go.js":                          "[JavaScript+GO]",
		".js.php":                         "[JavaScript+PHP]",
		".js.erb":                         "[JavaScript+Rails]",
		".tex, .ltx":                      "[LaTeX]",
		".lisp, .cl, .clisp, .l, .mud, .el, .scm, .ss, .lsp, .fasl, .sld": "[Lisp]",
		".lua":                                 "[Lua]",
		".matlab":                              "[MATLAB]",
		".mk, .mak, .make, .makefile, .mkfile": "[Makefile]",
		".gomd, .go.md, .hugo":                 "[Markdown+Go]",
		".ml, .mli, .mll, .mly":                "[OCamlyacc]",
		".m":                                   "[Objective-C]",
		".mm, .M":                              "[Objective-C++]",
		".php, .php3, .php4, .php5, .php7, .php8, .phps, .phpt, .aw, .ctp": "[PHP]",
		".phtml": "[PHP+HTML]",
		".txt":   "[Plain Text]",
		".py, .py3, .pyw, .pyi, .pyx, .pyx.in, .pxd, .pxd.in, .pxi, .pxi.in, .rpy, .cpy, .gyp, .gypi, .vpy, .smk, .wscript, .bazel, .bzl, .lmi, .pyde, .pyp, .pyt, .tac, .wsgi, .xpy": "[Python]",
		".R":  "[R]",
		".rd": "[Rd]",
		".re": "[R]",
		".rb": "[Regular Expression]",
		".rbi, .rbx, .rjs, .rabl, .rake, .capfile, .jbuilder, .gemspec, .podspec, .irbrc, .pryrc, .prawn, .thor, .god, .mspec, .pluginspec, .rbuild, .rbw, .ru, .ruby, .watchr": "[Ruby]",
		".ruby.rail, .rxml, .builder, .arb":              "[Ruby & Rails]",
		".rs, .rs.in":                                    "[Rust]",
		".sql, .ddl, .dml, .cql, .prc, .tab, .udf, .viw": "[SQL]",
		".sql.erb, .erbsql":                              "[SQL+Rails]",
		".scala, .sbt, .sc":                              "[Scala]",
		".ins":                                           "[TeX+DocStrip]",
		".textile":                                       "[Textile]",
		".ts":                                            "[TypeScript]",
		".xml, .tld, .dtml, .rng, .rss, .opml, .svg, .xaml": "[XML]",
		".xsd, .xsl, .xslt": "[XSL]",
		".yaml, .yml":       "[YAML]",
		".rst, .rest":       "[reStructuredText]",
		".abap":             "[abap]",
		".asc":              "[asc]",
		".ampl":             "[ampl]",
		".g4":               "[g4]",
		".apib":             "[apib]",
		".apl":              "[apl]",
		".dyalog":           "[dyalog]",
		".asax":             "[asax]",
		".ascx":             "[ascx]",
		".ashx":             "[ashx]",
		".asmx":             "[asmx]",
		".aspx":             "[aspx]",
		".axd":              "[axd]",
		".dats":             "[dats]",
		".hats":             "[hats]",
		".sats":             "[sats]",
		".adb":              "[adb]",
		".ada":              "[ada]",
		".ads":              "[ads]",
		".agda":             "[agda]",
		".als":              "[als]",
		".apacheconf":       "[apacheconf]",
		".vhost":            "[vhost]",
		".scpt":             "[scpt]",
		".arc":              "[arc]",
		".ino":              "[ino]",
		".asciidoc":         "[asciidoc]",
		".adoc":             "[adoc]",
		".aj":               "[aj]",
		".asm":              "[asm]",
		".a51":              "[a51]",
		".inc":              "[inc]",
		".nasm":             "[nasm]",
		".aug":              "[aug]",
		".ahk":              "[ahk]",
		".ahkl":             "[ahkl]",
		".au3":              "[au3]",
		".awk":              "[awk]",
		".auk":              "[auk]",
		".gawk":             "[gawk]",
		".mawk":             "[mawk]",
		".nawk":             "[nawk]",
		".befunge":          "[befunge]",
		".bison":            "[bison]",
		".bb":               "[bb]",
		".decls":            "[decls]",
		".bmx":              "[bmx]",
		".bsv":              "[bsv]",
		".boo":              "[boo]",
		".b":                "[b]",
		".bf":               "[bf]",
		".brs":              "[brs]",
		".bro":              "[bro]",
		".cats":             "[cats]",
		".idc":              "[idc]",
		".w":                "[w]",
		".cake":             "[cake]",
		".cshtml":           "[cshtml]",
		".hpp":              "[hpp]",
		".tcc":              "[tcc]",
		".tpp":              "[tpp]",
		".c-objdump":        "[c-objdump]",
		".chs":              "[chs]",
		".clp":              "[clp]",
		".cmake":            "[cmake]",
		".cmake.in":         "[cmake.in]",
		".cob":              "[cob]",
		".cbl":              "[cbl]",
		".ccp":              "[ccp]",
		".cobol":            "[cobol]",
		".csv":              "[csv]",
		".capnp":            "[capnp]",
		".mss":              "[mss]",
		".ceylon":           "[ceylon]",
		".chpl":             "[chpl]",
		".ch":               "[ch]",
		".ck":               "[ck]",
		".cirru":            "[cirru]",
		".clw":              "[clw]",
		".icl":              "[icl]",
		".dcl":              "[dcl]",
		".click":            "[click]",
		".boot":             "[boot]",
		".cl2":              "[cl2]",
		".cljs.hl":          "[cljs.hl]",
		".cljscm":           "[cljscm]",
		".cljx":             "[cljx]",
		".hic":              "[hic]",
		".coffee":           "[coffee]",
		"._coffee":          "[_coffee]",
		".cjsx":             "[cjsx]",
		".cson":             "[cson]",
		".iced":             "[iced]",
		".cfm":              "[cfm]",
		".cfml":             "[cfml]",
		".cfc":              "[cfc]",
		".asd":              "[asd]",
		".ny":               "[ny]",
		".podsl":            "[podsl]",
		".sexp":             "[sexp]",
		".cps":              "[cps]",
		".coq":              "[coq]",
		".v":                "[v]",
		".cppobjdump":       "[cppobjdump]",
		".c++-objdump":      "[c++-objdump]",
		".c++objdump":       "[c++objdump]",
		".cpp-objdump":      "[cpp-objdump]",
		".cxx-objdump":      "[cxx-objdump]",
		".creole":           "[creole]",
		".cr":               "[cr]",
		".feature":          "[feature]",
		".cu":               "[cu]",
		".cuh":              "[cuh]",
		".cy":               "[cy]",
		".d-objdump":        "[d-objdump]",
		".com":              "[com]",
		".dm":               "[dm]",
		".zone":             "[zone]",
		".arpa":             "[arpa]",
		".darcspatch":       "[darcspatch]",
		".dpatch":           "[dpatch]",
		".dart":             "[dart]",
		".dockerfile":       "[dockerfile]",
		".djs":              "[djs]",
		".dylan":            "[dylan]",
		".dyl":              "[dyl]",
		".intr":             "[intr]",
		".lid":              "[lid]",
		".E":                "[E]",
		".ecl":              "[ecl]",
		".eclxml":           "[eclxml]",
		".sch":              "[sch]",
		".brd":              "[brd]",
		".epj":              "[epj]",
		".e":                "[e]",
		".ex":               "[ex]",
		".exs":              "[exs]",
		".elm":              "[elm]",
		".emacs":            "[emacs]",
		".emacs.desktop":    "[emacs.desktop]",
		".em":               "[em]",
		".emberscript":      "[emberscript]",
		".es":               "[es]",
		".xrl":              "[xrl]",
		".yrl":              "[yrl]",
		".fs":               "[fs]",
		".fsi":              "[fsi]",
		".fsx":              "[fsx]",
		".fx":               "[fx]",
		".flux":             "[flux]",
		".f90":              "[f90]",
		".f":                "[f]",
		".f03":              "[f03]",
		".f08":              "[f08]",
		".f77":              "[f77]",
		".f95":              "[f95]",
		".for":              "[for]",
		".fpp":              "[fpp]",
		".factor":           "[factor]",
		".fy":               "[fy]",
		".fancypack":        "[fancypack]",
		".fan":              "[fan]",
		".eam.fs":           "[eam.fs]",
		".fth":              "[fth]",
		".4th":              "[4th]",
		".forth":            "[forth]",
		".fr":               "[fr]",
		".frt":              "[frt]",
		".ftl":              "[ftl]",
		".g":                "[g]",
		".gco":              "[gco]",
		".gcode":            "[gcode]",
		".gms":              "[gms]",
		".gap":              "[gap]",
		".gd":               "[gd]",
		".gi":               "[gi]",
		".tst":              "[tst]",
		".s":                "[s]",
		".ms":               "[ms]",
		".glsl":             "[glsl]",
		".fp":               "[fp]",
		".frag":             "[frag]",
		".frg":              "[frg]",
		".fsh":              "[fsh]",
		".fshader":          "[fshader]",
		".geo":              "[geo]",
		".geom":             "[geom]",
		".glslv":            "[glslv]",
		".gshader":          "[gshader]",
		".shader":           "[shader]",
		".vert":             "[vert]",
		".vrx":              "[vrx]",
		".vsh":              "[vsh]",
		".vshader":          "[vshader]",
		".gml":              "[gml]",
		".kid":              "[kid]",
		".ebuild":           "[ebuild]",
		".eclass":           "[eclass]",
		".po":               "[po]",
		".pot":              "[pot]",
		".glf":              "[glf]",
		".gp":               "[gp]",
		".gnu":              "[gnu]",
		".gnuplot":          "[gnuplot]",
		".plot":             "[plot]",
		".plt":              "[plt]",
		".golo":             "[golo]",
		".gs":               "[gs]",
		".gst":              "[gst]",
		".gsx":              "[gsx]",
		".vark":             "[vark]",
		".grace":            "[grace]",
		".gf":               "[gf]",
		".graphql":          "[graphql]",
		".man":              "[man]",
		".1":                "[1]",
		".1in":              "[1in]",
		".1m":               "[1m]",
		".1x":               "[1x]",
		".2":                "[2]",
		".3":                "[3]",
		".3in":              "[3in]",
		".3m":               "[3m]",
		".3qt":              "[3qt]",
		".3x":               "[3x]",
		".4":                "[4]",
		".5":                "[5]",
		".6":                "[6]",
		".7":                "[7]",
		".8":                "[8]",
		".9":                "[9]",
		".me":               "[me]",
		".n":                "[n]",
		".rno":              "[rno]",
		".roff":             "[roff]",
		".grt":              "[grt]",
		".gtpl":             "[gtpl]",
		".gsp":              "[gsp]",
		".hcl":              "[hcl]",
		".tf":               "[tf]",
		".hlsl":             "[hlsl]",
		".fxh":              "[fxh]",
		".hlsli":            "[hlsli]",
		".html.hl":          "[html.hl]",
		".st":               "[st]",
		".xht":              "[xht]",
		".mustache":         "[mustache]",
		".jinja":            "[jinja]",
		".eex":              "[eex]",
		".erb.deface":       "[erb.deface]",
		".http":             "[http]",
		".haml.deface":      "[haml.deface]",
		".handlebars":       "[handlebars]",
		".hbs":              "[hbs]",
		".hb":               "[hb]",
		".hsc":              "[hsc]",
		".hx":               "[hx]",
		".hxsl":             "[hxsl]",
		".hy":               "[hy]",
		".pro":              "[pro]",
		".dlm":              "[dlm]",
		".ipf":              "[ipf]",
		".ini":              "[ini]",
		".cfg":              "[cfg]",
		".prefs":            "[prefs]",
		".irclog":           "[irclog]",
		".weechatlog":       "[weechatlog]",
		".idr":              "[idr]",
		".lidr":             "[lidr]",
		".ni":               "[ni]",
		".i7x":              "[i7x]",
		".iss":              "[iss]",
		".io":               "[io]",
		".ik":               "[ik]",
		".thy":              "[thy]",
		".ijs":              "[ijs]",
		".flex":             "[flex]",
		".jflex":            "[jflex]",
		".geojson":          "[geojson]",
		".lock":             "[lock]",
		".topojson":         "[topojson]",
		".json5":            "[json5]",
		".jsonld":           "[jsonld]",
		".jq":               "[jq]",
		".jade":             "[jade]",
		".j":                "[j]",
		".js, .mjs, .cjs, .htc, .javascript, ._js, .bones, .es6, .jake, .jsb, .jscad, .jsfl, .jsm, .jss, .njs, .pac, .sjs, .ssjs, .xsjs, .xsjslib": "[JavaScript]",
		".jl":                                   "[Julia]",
		".ipynb":                                "[Jupyter Notebook]",
		".krl":                                  "[KRL]",
		".kicad_pcb":                            "[KiCad]",
		".kit":                                  "[Kit]",
		".kt, .ktm, .kts":                       "[Kotlin]",
		".lfe":                                  "[LFE]",
		".ll":                                   "[LLVM]",
		".lol":                                  "[LOLCODE]",
		".lsl, .lslp":                           "[LSL]",
		".lvproj":                               "[LabVIEW]",
		".lasso, .las, .lasso8, .lasso9, .ldml": "[Lasso]",
		".latte":                                "[Latte]",
		".lean, .hlean":                         "[Lean]",
		".less":                                 "[Less]",
		".lex":                                  "[Lex]",
		".ly, .ily":                             "[LilyPond]",
		".ld, .lds":                             "[Linker Script]",
		".liquid":                               "[Liquid]",
		".lagda":                                "[Literate Agda]",
		".litcoffee":                            "[Literate CoffeeScript]",
		".lhs":                                  "[Literate Haskell]",
		".ls, ._ls":                             "[LiveScript]",
		".xm, .x, .xi":                          "[Logos]",
		".lgt, .logtalk":                        "[Logtalk]",
		".lookml":                               "[LookML]",
		".fcgi, .nse, .pd_lua, .rbxs, .wlua":    "[Lua]",
		".mumps":                                "[M]",
		".m4":                                   "[M4/M4Sugar]",
		".mcr":                                  "[MAXScript]",
		".mtml":                                 "[MTML]",
		".muf":                                  "[MUF]",
		".mako, .mao":                           "[Mako]",
		".md, .mdown, .mdwn, .markdown, .markdn, .mkd, .mkdn, .mkdown, .ron": "[Markdown]",
		".mask": "[Mask]",
		".mathematica, .cdf, .ma, .mt, .nb, .nbp, .wl, .wlt": "[Mathematica]",
		".maxpat, .maxhelp, .maxproj, .mxt, .pat":            "[Max]",
		".mediawiki, .wiki":                                  "[MediaWiki]",
		".moo":                                               "[Mercury]",
		".metal":                                             "[Metal]",
		".minid":                                             "[MiniD]",
		".druby, .duby, .mir, .mirah":                        "[Mirah]",
		".mo":                                                "[Modelica]",
		".mms, .mmk":                                         "[Module Management System]",
		".monkey":                                            "[Monkey]",
		".moon":                                              "[MoonScript]",
		".myt":                                               "[Myghty]",
		".ncl":                                               "[NCL]",
		".nl":                                                "[NL]",
		".nsi, .nsh":                                         "[NSIS]",
		".axs, .axi":                                         "[NetLinx]",
		".axs.erb, .axi.erb":                                 "[NetLinx+ERB]",
		".nlogo":                                             "[NetLogo]",
		".nginxconf":                                         "[Nginx]",
		".nim, .nimrod":                                      "[Nimrod]",
		".ninja":                                             "[Ninja]",
		".nit, .nix":                                         "[Nit]",
		".nu":                                                "[Nu]",
		".numpy, .numpyw, .numsc":                            "[NumPy]",
		".eliom, .eliomi, .ml4":                              "[OCaml]",
		".objdump":                                           "[ObjDump]",
		".sj":                                                "[Objective-J]",
		".omgrofl":                                           "[Omgrofl]",
		".opa":                                               "[Opa]",
		".opal":                                              "[Opal]",
		".opencl":                                            "[OpenCL]",
		".scad":                                              "[OpenSCAD]",
		".org":                                               "[Org]",
		".ox, .oxh, .oxo":                                    "[Ox]",
		".oxygene":                                           "[Oxygene]",
		".oz":                                                "[Oz]",
		".pwn":                                               "[PAWN]",
		".pls, .pck, .pkb, .pks, .plb, .plsql":               "[PLpgSQL]",
		".pov":                                               "[POV-Ray SDL]",
		".pan":                                               "[Pan]",
		".psc":                                               "[Papyrus]",
		".parrot":                                            "[Parrot]",
		".pasm":                                              "[Parrot Assembly]",
		".pir":                                               "[Parrot Internal Representation]",
		".dfm, .lpr, .pp, .pas, .p, .dpr, .pascal":                          "[Pascal]",
		".pl, .pc, .al, .pm, .pmc, .pod, .t, .cgi, .perl, .ph, .plx, .psgi": "[Perl]",
		".6pl, .6pm, .nqp, .p6, .p6l, .p6m, .pl6, .pm6":                     "[Perl6]",
		".pkl":               "[Pickle]",
		".pig":               "[PigLatin]",
		".pike, .pmod":       "[Pike]",
		".pogo":              "[PogoScript]",
		".pony":              "[Pony]",
		".eps":               "[PostScript]",
		".ps1, .psd1, .psm1": "[PowerShell]",
		".pde":               "[Processing]",
		".prolog, .yap":      "[Prolog]",
		".spin":              "[Propeller Spin]",
		".proto":             "[Protocol Buffer]",
		".pub":               "[Public Key]",
		".pd":                "[Pure Data]",
		".pb, .pbi":          "[PureBasic]",
		".purs":              "[PureScript]",
		".pytb":              "[Python traceback]",
		".qml":               "[QML]",
		".qbs":               "[QML]",
		".pri":               "[QMake]",
		".r":                 "[R]",
		".rsx":               "[R]",
		".raml":              "[RAML]",
		".rdoc":              "[RDoc]",
		".rbbas, .rbfrm, .rbmnu, .rbres, .rbtbar, .rbuistate": "[REALbasic]",
		".rmd":                       "[RMarkdown]",
		".rkt, .rktd, .rktl, .scrbl": "[Racket]",
		".rl":                        "[Ragel in Ruby Host]",
		".raw":                       "[Raw token data]",
		".reb, .r2, .r3, .rebol":     "[Rebol]",
		".red, .reds":                "[Red]",
		".cw":                        "[Redcode]",
		".rsh":                       "[RenderScript]",
		".robot":                     "[RobotFramework]",
		".rg":                        "[Rouge]",
		".sas":                       "[SCSS]",
		".scss":                      "[scss]",
		".smt2, .smt":                "[SMT]",
		".sparql, .rq":               "[SPARQL]",
		".sqf, .hqf":                 "[SQF]",
		".db2":                       "[SQLPL]",
		".ston":                      "[STON]",
		".sage":                      "[Sage]",
		".sagews":                    "[Sage]",
		".sls":                       "[SaltStack]",
		".sass":                      "[Sass]",
		".scaml":                     "[Scaml]",
		".sps":                       "[Scheme]",
		".sci, .sce":                 "[Scilab]",
		".self":                      "[Self]",
		".sh-session":                "[ShellSession]",
		".shen":                      "[Shen]",
		".sl":                        "[Slash]",
		".slim":                      "[Slim]",
		".smali":                     "[Smali]",
		".tpl":                       "[Smarty]",
		".sp, .sma":                  "[SourcePawn]",
		".nut":                       "[Squirrel]",
		".stan":                      "[Stan]",
		".ML, .fun, .sig, .sml":      "[Standard ML]",
		".do, .ado, .doh, .ihlp, .mata, .matah, .sthlp": "[Stata]",
		".styl":          "[Stylus]",
		".scd":           "[SuperCollider]",
		".swift":         "[Swift]",
		".sv, .svh, .vh": "[SystemVerilog]",
		".toml":          "[TOML]",
		".txl":           "[TXL]",
		".tm":            "[Tcl]",
		".tcsh, .csh":    "[Tcsh]",
		".tea":           "[Tea]",
		".no":            "[Text]",
		".thrift":        "[Thrift]",
		".tu":            "[Turing]",
		".ttl":           "[Turtle]",
		".twig":          "[Twig]",
		".upc":           "[Unified Parallel C]",
		".anim, .asset, .mat, .meta, .prefab, .unity": "[Unity3D Asset]",
		".uno":      "[Uno]",
		".uc":       "[UnrealScript]",
		".ur, .urs": "[UrWeb]",
		".vcl":      "[VCL]",
		".vhdl, .vhd, .vhf, .vhi, .vho, .vhs, .vht, .vhw": "[VHDL]",
		".vala, .vapi":                         "[Vala]",
		".veo":                                 "[Verilog]",
		".vim":                                 "[VimL]",
		".vb, .bas, .frm, .frx, .vba, .vbhtml": "[Visual Basic]",
		".volt":                                "[Volt]",
		".vue":                                 "[Vue]",
		".owl":                                 "[owl]",
		".webidl":                              "[webidl]",
		".x10":                                 "[x10]",
		".xc":                                  "[xc]",
		".ant":                                 "[ant]",
		".axml":                                "[axml]",
		".ccxml":                               "[ccxml]",
		".clixml":                              "[clixml]",
		".cproject":                            "[cproject]",
		".csl":                                 "[csl]",
		".csproj":                              "[csproj]",
		".ct":                                  "[ct]",
		".dita":                                "[dita]",
		".ditamap":                             "[ditamap]",
		".ditaval":                             "[ditaval]",
		".dll.config":                          "[dll.config]",
		".dotsettings":                         "[dotsettings]",
		".filters":                             "[filters]",
		".fsproj":                              "[fsproj]",
		".fxml":                                "[fxml]",
		".glade":                               "[glade]",
		".grxml":                               "[grxml]",
		".iml":                                 "[iml]",
		".ivy":                                 "[ivy]",
		".jelly":                               "[jelly]",
		".jsproj":                              "[jsproj]",
		".kml":                                 "[kml]",
		".launch":                              "[launch]",
		".mdpolicy":                            "[mdpolicy]",
		".mxml":                                "[mxml]",
		".nproj":                               "[nproj]",
		".nuspec":                              "[nuspec]",
		".odd":                                 "[odd]",
		".osm":                                 "[osm]",
		".plist":                               "[plist]",
		".props":                               "[props]",
		".ps1xml":                              "[ps1xml]",
		".psc1":                                "[psc1]",
		".pt":                                  "[pt]",
		".rdf":                                 "[rdf]",
		".scxml":                               "[scxml]",
		".srdf":                                "[srdf]",
		".storyboard":                          "[storyboard]",
		".stTheme":                             "[stTheme]",
		".targets":                             "[targets]",
		".tmCommand":                           "[tmCommand]",
		".tml":                                 "[tml]",
		".tmLanguage":                          "[tmLanguage]",
		".tmPreferences":                       "[tmPreferences]",
		".tmSnippet":                           "[tmSnippet]",
		".tmTheme":                             "[tmTheme]",
		".ui":                                  "[ui]",
		".urdf":                                "[urdf]",
		".ux":                                  "[ux]",
		".vbproj":                              "[vbproj]",
		".vcxproj":                             "[vcxproj]",
		".vssettings":                          "[vssettings]",
		".vxml":                                "[vxml]",
		".wsdl":                                "[wsdl]",
		".wsf":                                 "[wsf]",
		".wxi":                                 "[wxi]",
		".wxl":                                 "[wxl]",
		".wxs":                                 "[wxs]",
		".x3d":                                 "[x3d]",
		".xacro":                               "[xacro]",
		".xib":                                 "[xib]",
		".xlf":                                 "[xlf]",
		".xliff":                               "[xliff]",
		".xmi":                                 "[xmi]",
		".xml.dist":                            "[xml.dist]",
		".xproj":                               "[xproj]",
		".xul":                                 "[xul]",
		".zcml":                                "[zcml]",
		".xsp-config":                          "[xsp-config]",
		".xsp.metadata":                        "[xsp.metadata]",
		".xpl":                                 "[xpl]",
		".xproc":                               "[xproc]",
		".xquery":                              "[xquery]",
		".xq":                                  "[xq]",
		".xql":                                 "[xql]",
		".xqm":                                 "[xqm]",
		".xqy":                                 "[xqy]",
		".xs":                                  "[xs]",
		".xojo_code":                           "[xojo_code]",
		".xojo_menu":                           "[xojo_menu]",
		".xojo_report":                         "[xojo_report]",
		".xojo_script":                         "[xojo_script]",
		".xojo_toolbar":                        "[xojo_toolbar]",
		".xojo_window":                         "[xojo_window]",
		".xtend":                               "[xtend]",
		".reek":                                "[reek]",
		".rviz":                                "[rviz]",
		".syntax":                              "[syntax]",
		".yaml-tmlanguage":                     "[yaml-tmlanguage]",
		".yang":                                "[yang]",
		".y":                                   "[y]",
		".yacc":                                "[yacc]",
		".yy":                                  "[yy]",
		".zep":                                 "[zep]",
		".zimpl":                               "[zimpl]",
		".zmpl":                                "[zmpl]",
		".zpl":                                 "[zpl]",
		".desktop":                             "[desktop]",
		".desktop.in":                          "[desktop.in]",
		".ec":                                  "[ec]",
		".eh":                                  "[eh]",
		".fish":                                "[fish]",
		".mu":                                  "[mu]",
		".nc":                                  "[nc]",
		".ooc":                                 "[ooc]",
		".rest.txt":                            "[rest.txt]",
		".rst.txt":                             "[rst.txt]",
		".wisp":                                "[wisp]",
		".prg":                                 "[prg]",
		".prw":                                 "[prw]",
		".bsl":                                 "[bsl]",
		".os":                                  "[os]",
		".2da":                                 "[2da]",
		".4dm":                                 "[4dm]",
		".asddls":                              "[asddls]",
		".abnf":                                "[abnf]",
		".aidl":                                "[aidl]",
		".asl":                                 "[asl]",
		".dsl":                                 "[dsl]",
		".asn":                                 "[asn]",
		".asn1":                                "[asn1]",
		".afm":                                 "[afm]",
		".OutJob":                              "[OutJob]",
		".PcbDoc":                              "[PcbDoc]",
		".PrjPCB":                              "[PrjPCB]",
		".SchDoc":                              "[SchDoc]",
		".angelscript":                         "[angelscript]",
		".antlers.html":                        "[antlers.html]",
		".antlers.php":                         "[antlers.php]",
		".antlers.xml":                         "[antlers.xml]",
		".trigger":                             "[trigger]",
		".agc":                                 "[agc]",
		".i":                                   "[i]",
		".nas":                                 "[nas]",
		".astro":                               "[astro]",
		".asy":                                 "[asy]",
		".avdl":                                "[avdl]",
		".bqn":                                 "[bqn]",
		".bal":                                 "[bal]",
		".be":                                  "[be]",
		".bibtex":                              "[bibtex]",
		".bicep":                               "[bicep]",
		".bicepparam":                          "[bicepparam]",
		".bs":                                  "[bs]",
		".bbappend":                            "[bbappend]",
		".bbclass":                             "[bbclass]",
		".blade":                               "[blade]",
		".blade.php":                           "[blade.php]",
		".bpl":                                 "[bpl]",
		".cs.pp":                               "[cs.pp]",
		".linq":                                "[linq]",
		".txx":                                 "[txx]",
		".cds":                                 "[cds]",
		".cil":                                 "[cil]",
		".dae":                                 "[dae]",
		".cue":                                 "[cue]",
		".caddyfile":                           "[caddyfile]",
		".cdc":                                 "[cdc]",
		".cairo":                               "[cairo]",
		".mligo":                               "[mligo]",
		".carbon":                              "[carbon]",
		".crc32":                               "[crc32]",
		".md2":                                 "[md2]",
		".md4":                                 "[md4]",
		".md5":                                 "[md5]",
		".sha1":                                "[sha1]",
		".sha2":                                "[sha2]",
		".sha224":                              "[sha224]",
		".sha256":                              "[sha256]",
		".sha256sum":                           "[sha256sum]",
		".sha3":                                "[sha3]",
		".sha384":                              "[sha384]",
		".sha512":                              "[sha512]",
		".circom":                              "[circom]",
		".clar":                                "[clar]",
		".soy":                                 "[soy]",
		".conllu":                              "[conllu]",
		".conll":                               "[conll]",
		".ql":                                  "[ql]",
		".qll":                                 "[qll]",
		".cwl":                                 "[cwl]",
		".orc":                                 "[orc]",
		".udo":                                 "[udo]",
		".csd":                                 "[csd]",
		".sco":                                 "[sco]",
		".curry":                               "[curry]",
		".cylc":                                "[cylc]",
		".cyp":                                 "[cyp]",
		".cypher":                              "[cypher]",
		".d2":                                  "[d2]",
		".dfy":                                 "[dfy]",
		".dwl":                                 "[dwl]",
		".dsc":                                 "[dsc]",
		".dhall":                               "[dhall]",
		".env":                                 "[env]",
		".eml":                                 "[eml]",
		".mbox":                                "[mbox]",
		".ebnf":                                "[ebnf]",
		".ejs":                                 "[ejs]",
		".ect":                                 "[ect]",
		".ejs.t":                               "[ejs.t]",
		".jst":                                 "[jst]",
		".eq":                                  "[eq]",
		".eb":                                  "[eb]",
		".edge":                                "[edge]",
		".edgeql":                              "[edgeql]",
		".esdl":                                "[esdl]",
		".editorconfig":                        "[editorconfig]",
		".edc":                                 "[edc]",
		".elv":                                 "[elv]",
		".app":                                 "[app]",
		".app.src":                             "[app.src]",
		".fst":                                 "[fst]",
		".fsti":                                "[fsti]",
		".flf":                                 "[flf]",
		".fir":                                 "[fir]",
		".dsp":                                 "[dsp]",
		".fnl":                                 "[fnl]",
		".bi":                                  "[bi]",
		".fut":                                 "[fut]",
		".cnc":                                 "[cnc]",
		".gaml":                                "[gaml]",
		".gdb":                                 "[gdb]",
		".gdbinit":                             "[gdbinit]",
		".ged":                                 "[ged]",
		".glslf":                               "[glslf]",
		".rchit":                               "[rchit]",
		".rmiss":                               "[rmiss]",
		".tesc":                                "[tesc]",
		".tese":                                "[tese]",
		".vs":                                  "[vs]",
		".gn":                                  "[gn]",
		".gni":                                 "[gni]",
		".gsc":                                 "[gsc]",
		".csc":                                 "[csc]",
		".gsh":                                 "[gsh]",
		".gmi":                                 "[gmi]",
		".4gl":                                 "[4gl]",
		".per":                                 "[per]",
		".gbr":                                 "[gbr]",
		".cmp":                                 "[cmp]",
		".gbl":                                 "[gbl]",
		".gbo":                                 "[gbo]",
		".gbp":                                 "[gbp]",
		".gbs":                                 "[gbs]",
		".gko":                                 "[gko]",
		".gpb":                                 "[gpb]",
		".gpt":                                 "[gpt]",
		".gtl":                                 "[gtl]",
		".gto":                                 "[gto]",
		".gtp":                                 "[gtp]",
		".gts":                                 "[gts]",
		".sol":                                 "[sol]",
		".story":                               "[story]",
		".gleam":                               "[gleam]",
		".gjs":                                 "[gjs]",
		".bdf":                                 "[bdf]",
		".gdnlib":                              "[gdnlib]",
		".gdns":                                "[gdns]",
		".tres":                                "[tres]",
		".tscn":                                "[tscn]",
		".gradle.kts":                          "[gradle.kts]",
		".gql":                                 "[gql]",
		".graphqls":                            "[graphqls]",
		".nomad":                               "[nomad]",
		".tfvars":                              "[tfvars]",
		".workflow":                            "[workflow]",
		".cginc":                               "[cginc]",
		".hocon":                               "[hocon]",
		".hta":                                 "[hta]",
		".ecr":                                 "[ecr]",
		".html.heex":                           "[html.heex]",
		".html.leex":                           "[html.leex]",
		".razor":                               "[razor]",
		".hxml":                                "[hxml]",
		".hack":                                "[hack]",
		".hhi":                                 "[hhi]",
		".q":                                   "[q]",
		".hql":                                 "[hql]",
		".hc":                                  "[hc]",
		".cnf":                                 "[cnf]",
		".dof":                                 "[dof]",
		".lektorproject":                       "[lektorproject]",
		".url":                                 "[url]",
		".ijm":                                 "[ijm]",
		".imba":                                "[imba]",
		".ink":                                 "[ink]",
		".isl":                                 "[isl]",
		".jcl":                                 "[jcl]",
		".4DForm":                              "[4DForm]",
		".4DProject":                           "[4DProject]",
		".avsc":                                "[avsc]",
		".gltf":                                "[gltf]",
		".har":                                 "[har]",
		".ice":                                 "[ice]",
		".JSON-tmLanguage":                     "[JSON-tmLanguage]",
		".jsonl":                               "[jsonl]",
		".mcmeta":                              "[mcmeta]",
		".sarif":                               "[sarif]",
		".tfstate":                             "[tfstate]",
		".tfstate.backup":                      "[tfstate.backup]",
		".webapp":                              "[webapp]",
		".webmanifest":                         "[webmanifest]",
		".yyp":                                 "[yyp]",
		".code-snippets":                       "[code-snippets]",
		".code-workspace":                      "[code-workspace]",
		".janet":                               "[janet]",
		".jav":                                 "[jav]",
		".jsh":                                 "[jsh]",
		".tag":                                 "[tag]",
		".jte":                                 "[jte]",
		".jslib":                               "[jslib]",
		".jspre":                               "[jspre]",
		".snap":                                "[snap]",
		".mps":                                 "[mps]",
		".mpl":                                 "[mpl]",
		".msd":                                 "[msd]",
		".j2":                                  "[j2]",
		".jinja2":                              "[jinja2]",
		".jison":                               "[jison]",
		".jisonlex":                            "[jisonlex]",
		".ol":                                  "[ol]",
		".iol":                                 "[iol]",
		".jsonnet":                             "[jsonnet]",
		".libsonnet":                           "[libsonnet]",
		".just":                                "[just]",
		".ksy":                                 "[ksy]",
		".kak":                                 "[kak]",
		".ks":                                  "[ks]",
		".kicad_mod":                           "[kicad_mod]",
		".kicad_wks":                           "[kicad_wks]",
		".kicad_sch":                           "[kicad_sch]",
		".kql":                                 "[kql]",
		".lvclass":                             "[lvclass]",
		".lvlib":                               "[lvlib]",
		".lark":                                "[lark]",
		".ligo":                                "[ligo]",
		".coffee.md":                           "[coffee.md]",
		".livecodescript":                      "[livecodescript]",
		".lkml":                                "[lkml]",
		".p8":                                  "[p8]",
		".rockspec":                            "[rockspec]",
		".luau":                                "[luau]",
		".mc":                                  "[mc]",
		".mdx":                                 "[mdx]",
		".mlir":                                "[mlir]",
		".mq4":                                 "[mq4]",
		".mqh":                                 "[mqh]",
		".mq5":                                 "[mq5]",
		".m2":                                  "[m2]",
		".livemd":                              "[livemd]",
		".ronn":                                "[ronn]",
		".workbook":                            "[workbook]",
		".marko":                               "[marko]",
		".mmd":                                 "[mmd]",
		".mermaid":                             "[mermaid]",
		".sln":                                 "[sln]",
		".mint":                                "[mint]",
		".i3":                                  "[i3]",
		".ig":                                  "[ig]",
		".m3":                                  "[m3]",
		".mg":                                  "[mg]",
		".mojo":                                "[mojo]",
		".monkey2":                             "[monkey2]",
		".x68":                                 "[x68]",
		".move":                                "[move]",
		".muse":                                "[muse]",
		".nasl":                                "[nasl]",
		".neon":                                "[neon]",
		".nss":                                 "[nss]",
		".ne":                                  "[ne]",
		".nearley":                             "[nearley]",
		".nf":                                  "[nf]",
		".nginx":                               "[nginx]",
		".nim.cfg":                             "[nim.cfg]",
		".nimble":                              "[nimble]",
		".nims":                                "[nims]",
		".nr":                                  "[nr]",
		".njk":                                 "[njk]",
		".ob2":                                 "[ob2]",
		".odin":                                "[odin]",
		".rego":                                "[rego]",
		".qasm":                                "[qasm]",
		".glyphs":                              "[glyphs]",
		".fea":                                 "[fea]",
		".p4":                                  "[p4]",
		".pddl":                                "[pddl]",
		".pegjs":                               "[pegjs]",
		".peggy":                               "[peggy]",
		".bdy":                                 "[bdy]",
		".fnc":                                 "[fnc]",
		".spc":                                 "[spc]",
		".tpb":                                 "[tpb]",
		".tps":                                 "[tps]",
		".trg":                                 "[trg]",
		".vw":                                  "[vw]",
		".pgsql":                               "[pgsql]",
		".pact":                                "[pact]",
		".pep":                                 "[pep]",
		".pic":                                 "[pic]",
		".chem":                                "[chem]",
		".puml":                                "[puml]",
		".iuml":                                "[iuml]",
		".plantuml":                            "[plantuml]",
		".pod6":                                "[pod6]",
		".polar":                               "[polar]",
		".por":                                 "[por]",
		".pcss":                                "[pcss]",
		".postcss":                             "[postcss]",
		".epsi":                                "[epsi]",
		".pfa":                                 "[pfa]",
		".pbt":                                 "[pbt]",
		".sra":                                 "[sra]",
		".sru":                                 "[sru]",
		".srw":                                 "[srw]",
		".praat":                               "[praat]",
		".prisma":                              "[prisma]",
		".pml":                                 "[pml]",
		".textproto":                           "[textproto]",
		".pbtxt":                               "[pbtxt]",
		".pug":                                 "[pug]",
		".arr":                                 "[arr]",
		".spec":                                "[spec]",
		".qs":                                  "[qs]",
		".rbs":                                 "[rbs]",
		".rexx":                                "[rexx]",
		".pprx":                                "[pprx]",
		".rex":                                 "[rex]",
		".qmd":                                 "[qmd]",
		".rpgle":                               "[rpgle]",
		".sqlrpgle":                            "[sqlrpgle]",
		".rnh":                                 "[rnh]",
		".raku":                                "[raku]",
		".rakumod":                             "[rakumod]",
		".rsc":                                 "[rsc]",
		".res":                                 "[res]",
		".rei":                                 "[rei]",
		".religo":                              "[religo]",
		".regexp":                              "[regexp]",
		".regex":                               "[regex]",
		".ring":                                "[ring]",
		".riot":                                "[riot]",
		".resource":                            "[resource]",
		".roc":                                 "[roc]",
		".3p":                                  "[3p]",
		".3pm":                                 "[3pm]",
		".mdoc":                                "[mdoc]",
		".tmac":                                "[tmac]",
		".eye":                                 "[eye]",
		".te":                                  "[te]",
		".mysql":                               "[mysql]",
		".srt":                                 "[srt]",
		".star":                                "[star]",
		".stl":                                 "[stl]",
		".kojo":                                "[kojo]",
		".scenic":                              "[scenic]",
		".zsh-theme":                           "[zsh-theme]",
		".sieve":                               "[sieve]",
		".sfv":                                 "[sfv]",
		".slint":                               "[slint]",
		".cocci":                               "[cocci]",
		".smithy":                              "[smithy]",
		".snakefile":                           "[snakefile]",
		".sfd":                                 "[sfd]",
		".sss":                                 "[sss]",
		".svelte":                              "[svelte]",
		".sw":                                  "[sw]",
		".rnw":                                 "[rnw]",
		".8xp":                                 "[8xp]",
		".8xp.txt":                             "[8xp.txt]",
		".tlv":                                 "[tlv]",
		".tla":                                 "[tla]",
		".tsv":                                 "[tsv]",
		".vcf":                                 "[vcf]",
		".talon":                               "[talon]",
		".sdc":                                 "[sdc]",
		".tcl.in":                              "[tcl.in]",
		".xdc":                                 "[xdc]",
		".tftpl":                               "[tftpl]",
		".texinfo":                             "[texinfo]",
		".texi":                                "[texi]",
		".txi":                                 "[txi]",
		".TextGrid":                            "[TextGrid]",
		".toit":                                "[toit]",
		".tl":                                  "[tl]",
		".cts":                                 "[cts]",
		".mts":                                 "[mts]",
		".typ":                                 "[typ]",
		".vdf":                                 "[vdf]",
		".vtl":                                 "[vtl]",
		".vimrc":                               "[vimrc]",
		".vmb":                                 "[vmb]",
		".snip":                                "[snip]",
		".snippet":                             "[snippet]",
		".snippets":                            "[snippets]",
		".ctl":                                 "[ctl]",
		".Dsr":                                 "[Dsr]",
		".vy":                                  "[vy]",
		".wdl":                                 "[wdl]",
		".wgsl":                                "[wgsl]",
		".mtl":                                 "[mtl]",
		".obj":                                 "[obj]",
		".wast":                                "[wast]",
		".wat":                                 "[wat]",
		".wit":                                 "[wit]",
		".vtt":                                 "[vtt]",
		".whiley":                              "[whiley]",
		".wikitext":                            "[wikitext]",
		".reg":                                 "[reg]",
		".ws":                                  "[ws]",
		".wlk":                                 "[wlk]",
		".wren":                                "[wren]",
		".xbm":                                 "[xbm]",
		".xpm":                                 "[xpm]",
		".adml":                                "[adml]",
		".admx":                                "[admx]",
		".axaml":                               "[axaml]",
		".builds":                              "[builds]",
		".ccproj":                              "[ccproj]",
		".cscfg":                               "[cscfg]",
		".csdef":                               "[csdef]",
		".depproj":                             "[depproj]",
		".gmx":                                 "[gmx]",
		".hzp":                                 "[hzp]",
		".mjml":                                "[mjml]",
		".natvis":                              "[natvis]",
		".ndproj":                              "[ndproj]",
		".pkgproj":                             "[pkgproj]",
		".proj":                                "[proj]",
		".qhelp":                               "[qhelp]",
		".resx":                                "[resx]",
		".sfproj":                              "[sfproj]",
		".shproj":                              "[shproj]",
		".vsixmanifest":                        "[vsixmanifest]",
		".vstemplate":                          "[vstemplate]",
		".wixproj":                             "[wixproj]",
		".xmp":                                 "[xmp]",
		".xspec":                               "[xspec]",
		".xsh":                                 "[xsh]",
		".yaml.sed":                            "[yaml.sed]",
		".yml.mysql":                           "[yml.mysql]",
		".yar":                                 "[yar]",
		".yara":                                "[yara]",
		".yasnippet":                           "[yasnippet]",
		".yul":                                 "[yul]",
		".zap":                                 "[zap]",
		".xzap":                                "[xzap]",
		".zil":                                 "[zil]",
		".zeek":                                "[zeek]",
		".zs":                                  "[zs]",
		".zig":                                 "[zig]",
		".zig.zon":                             "[zig.zon]",
		".service":                             "[service]",
		".dircolors":                           "[dircolors]",
		".hoon":                                "[hoon]",
		".ics":                                 "[ics]",
		".ical":                                "[ical]",
		".kv":                                  "[kv]",
		".mrc":                                 "[mrc]",
		".mcfunction":                          "[mcfunction]",
		".nanorc":                              "[nanorc]",
		".sed":                                 "[sed]",
		".templ":                               "[templ]",
	}

	// Check if the URL ends with one of the passive extensions
	for extGroup, label := range passiveExtensions {
		// Split the extensions into a slice
		extensions := strings.Split(extGroup, ", ")
		for _, ext := range extensions {
			if strings.HasSuffix(url, ext) {
				if passive {
					// If passive mode is on, just print the URL and its label.
					if jsonOutput {
						output := JSONOutput{
							Host: url,
							Type: "EXTENSION BASED",
						}
						output.Data.Suffix = strings.Trim(label, "[]") // Remove both brackets

						var jsonData []byte
						if jsonTypeFlag == "Marshal" {
							jsonData, _ = json.Marshal(output)
						} else {
							jsonData, _ = json.MarshalIndent(output, "", "  ") // Pretty print the JSON
						}
						fmt.Println(string(jsonData))
						if outputFile != nil {
							outputFile.WriteString(string(jsonData) + "\n")
						}
						return
					}

					// Handle non-verbose and verbose output.
					outputLine := ""
					if verbose {
						if options.NoColor {
							outputLine = fmt.Sprintf("EXTENSION BASED: %s %s\n", url, label)
						} else {
							outputLine = fmt.Sprintf("%s: %s %s\n", aurora.Cyan("EXTENSION BASED"), url, aurora.Yellow(label))
						}
					} else {
						if options.NoColor {
							outputLine = fmt.Sprintf("%s %s\n", url, label)
						} else {
							outputLine = fmt.Sprintf("%s %s\n", url, aurora.Yellow(label))
						}
					}
					fmt.Print(outputLine)
					if outputFile != nil {
						outputFile.WriteString(outputLine)
					}

					time.Sleep(delay) // Apply delay between requests
					return
				}
			}
		}
	}

	// If not passive or extension not in map, proceed with the request.
	getURLInfo(url, verbose, timeout, insecure, userAgent, jsonOutput, jsonTypeFlag, outputFile, options)
	time.Sleep(delay) // Apply delay between requests
}

func main() {
	// Command-line flags
	options := ParseOptions()

	if options.Version {
		banner.PrintBanner()
		banner.PrintVersion()
		return
	}

	if !options.Silent {
		banner.PrintBanner()
	}

	// Convert Timeout to a time.Duration
	timeout := time.Duration(options.Timeout) * time.Second

	// Set up a WaitGroup and a semaphore (channel) to control concurrency
	var wg sync.WaitGroup
	sem := make(chan struct{}, options.Threads)

	var outputFile *os.File
	var err error
	if options.Output != "" || options.AppendOutput != "" {
		var outputFileName string
		if options.AppendOutput != "" {
			outputFileName = options.AppendOutput
		} else {
			outputFileName = options.Output
		}

		outputFile, err = os.OpenFile(outputFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Printf("Error opening output file %s: %v\n", outputFileName, err)
			return
		}
		defer outputFile.Close()
	}

	if options.InputTargetHost != "" {
		wg.Add(1)
		go processURL(options.InputTargetHost, options.Passive, options.Verbose, timeout, options.Insecure, options.UserAgent, &wg, sem, options.Delay, options.JSONOutput, options.JSONtype, outputFile, options)
		wg.Wait()
		return
	}

	if options.InputFile != "" {
		file, err := os.Open(options.InputFile)
		if err != nil {
			fmt.Printf("Error opening file %s: %v\n", options.InputFile, err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			url := strings.TrimSpace(scanner.Text())
			if url != "" {
				wg.Add(1)
				go processURL(url, options.Passive, options.Verbose, timeout, options.Insecure, options.UserAgent, &wg, sem, options.Delay, options.JSONOutput, options.JSONtype, outputFile, options)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file: %v\n", err)
		}
		wg.Wait()
		return
	}

	// Read from stdin
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		url := strings.TrimSpace(scanner.Text())
		if url != "" {
			wg.Add(1)
			go processURL(url, options.Passive, options.Verbose, timeout, options.Insecure, options.UserAgent, &wg, sem, options.Delay, options.JSONOutput, options.JSONtype, outputFile, options)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading stdin: %v\n", err)
	}
	wg.Wait()
}
