## linkinspector

linkinspector is a command-line tool that analyzes URLs to retrieve HTTP status codes, content lengths, and content types. It features color-coded output, passive checks for specific file extensions, and supports input from stdin or files.

## Installation
```
go install github.com/rix4uni/linkinspector@latest
```

## Download prebuilt binaries
```
wget https://github.com/rix4uni/linkinspector/releases/download/v0.0.1/linkinspector-linux-amd64-0.0.1.tgz
tar -xvzf linkinspector-linux-amd64-0.0.1.tgz
rm -rf linkinspector-linux-amd64-0.0.1.tgz
mv linkinspector ~/go/bin/linkinspector
```
Or download [binary release](https://github.com/rix4uni/linkinspector/releases) for your platform.

## Compile from source
```
git clone --depth 1 github.com/rix4uni/linkinspector.git
cd linkinspector; go install
```

## Usage
```console
Usage of linkinspector:
  -ao string
        File to append output results instead of overwriting
  -c int
        Number of concurrent goroutines (default 50)
  -delay duration
        Duration between each HTTP request (eg: 200ms, 1s) (default -1ns)
  -insecure
        Disable TLS certificate verification
  -json
        Output in JSON format
  -json-type string
        Output in JSON type, MarshalIndent or Marshal (default "MarshalIndent")
  -list string
        File containing list of URLs to check
  -o string
        File to write output results
  -passive
        Enable passive mode to skip requests for specific extensions
  -silent
        silent mode.
  -timeout int
        HTTP request timeout duration (in seconds) (default 10)
  -u string
        Single URL to check
  -ua string
        Custom User-Agent header for HTTP requests (default "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")
  -verbose
        Enable verbose mode to print more information
  -version
        Print the version of the tool and exit.
```

## Usage Examples

#### Single URL
```bash
┌──(root㉿kali)-[/root/linkinspector]
└─# echo "https://linkinspector.netlify.app/nuclei-templates.zip" | linkinspector
```

#### Multiple URLs
```bash
┌──(root㉿kali)-[/root/linkinspector]
└─# cat urls.txt | linkinspector
```

## Supported types

#### Image

- **jpg** - `image/jpeg`
- **png** - `image/png`
- **gif** - `image/gif`
- **webp** - `image/webp`
- **cr2** - `image/x-canon-cr2`
- **tif** - `image/tiff`
- **bmp** - `image/bmp`
- **heif** - `image/heif`
- **jxr** - `image/vnd.ms-photo`
- **psd** - `image/vnd.adobe.photoshop`
- **ico** - `image/vnd.microsoft.icon`
- **dwg** - `image/vnd.dwg`
- **avif** - `image/avif`

#### Video

- **mp4** - `video/mp4`
- **m4v** - `video/x-m4v`
- **mkv** - `video/x-matroska`
- **webm** - `video/webm`
- **mov** - `video/quicktime`
- **avi** - `video/x-msvideo`
- **wmv** - `video/x-ms-wmv`
- **mpg** - `video/mpeg`
- **flv** - `video/x-flv`
- **3gp** - `video/3gpp`

#### Audio

- **mid** - `audio/midi`
- **mp3** - `audio/mpeg`
- **m4a** - `audio/mp4`
- **ogg** - `audio/ogg`
- **flac** - `audio/x-flac`
- **wav** - `audio/x-wav`
- **amr** - `audio/amr`
- **aac** - `audio/aac`
- **aiff** - `audio/x-aiff`

#### Archive

- **epub** - `application/epub+zip`
- **zip** - `application/zip`
- **tar** - `application/x-tar`
- **rar** - `application/vnd.rar`
- **gz** - `application/gzip`
- **bz2** - `application/x-bzip2`
- **7z** - `application/x-7z-compressed`
- **xz** - `application/x-xz`
- **zstd** - `application/zstd`
- **pdf** - `application/pdf`
- **exe** - `application/vnd.microsoft.portable-executable`
- **swf** - `application/x-shockwave-flash`
- **rtf** - `application/rtf`
- **iso** - `application/x-iso9660-image`
- **eot** - `application/octet-stream`
- **ps** - `application/postscript`
- **sqlite** - `application/vnd.sqlite3`
- **nes** - `application/x-nintendo-nes-rom`
- **crx** - `application/x-google-chrome-extension`
- **cab** - `application/vnd.ms-cab-compressed`
- **deb** - `application/vnd.debian.binary-package`
- **ar** - `application/x-unix-archive`
- **Z** - `application/x-compress`
- **lz** - `application/x-lzip`
- **rpm** - `application/x-rpm`
- **elf** - `application/x-executable`
- **dcm** - `application/dicom`

#### Documents

- **doc** - `application/msword`
- **docx** - `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- **xls** - `application/vnd.ms-excel`
- **xlsx** - `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`
- **ppt** - `application/vnd.ms-powerpoint`
- **pptx** - `application/vnd.openxmlformats-officedocument.presentationml.presentation`

#### Font

- **woff** - `application/font-woff`
- **woff2** - `application/font-woff`
- **ttf** - `application/font-sfnt`
- **otf** - `application/font-sfnt`

#### Application

- **wasm** - `application/wasm`
- **dex** - `application/vnd.android.dex`
- **dey** - `application/vnd.android.dey`

## TODO
add these flags

- -csv
  Output in CSV format

- -nc
  Do not write colored output to terminal or file

- use "github.com/projectdiscovery/goflags"

```
MATCHERS:
   -mc, -match-code string            match response with specified status code (-mc 200,302)
   -ml, -match-length string          match response with specified content length (-ml 100,102)
   -mct, -match-content-type string   match response with specified content type (-mct "application/octet-stream,text/html")
   -msn, -match-suffix-name string    match response with specified suffix name (-msn "ZIP,PHP,7Z")
```


```
FILTERS:
   -fc, -filter-code string            filter response with specified status code (-fc 403,401)
   -fl, -filter-length string          filter response with specified content length (-fl 23,33)
   -fct, -filter-content-type string   filter response with specified content type (-fct "text/html,image/jpeg")
   -fsn, -filter-suffix-name string    filter response with specified suffix name (-fsn "CSS,Plain Text,html")
```

## Extension Sources
- https://gist.github.com/ppisarczyk/43962d06686722d26d176fad46879d41
- https://github.com/github-linguist/linguist/blob/main/lib/linguist/languages.yml