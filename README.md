## linkinspector

linkinspector is a fast command-line tool for inspecting URLs and retrieving HTTP status codes, content lengths, and content types. It supports filtering and matching responses, and can process URLs from stdin or files.

## Installation
```
go install github.com/rix4uni/linkinspector@latest
```

## Download prebuilt binaries
```
wget https://github.com/rix4uni/linkinspector/releases/download/v0.0.5/linkinspector-linux-amd64-0.0.5.tgz
tar -xvzf linkinspector-linux-amd64-0.0.5.tgz
rm -rf linkinspector-linux-amd64-0.0.5.tgz
mv linkinspector ~/go/bin/linkinspector
```
Or download [binary release](https://github.com/rix4uni/linkinspector/releases) for your platform.

## Compile from source
```
git clone --depth 1 github.com/rix4uni/linkinspector.git
cd linkinspector; go install
```

## Configuration

linkinspector requires a `extensions.yaml` file for extension mappings. The tool automatically manages this file in your home directory.

### extensions.yaml Structure

The `extensions.yaml` file must contain two main sections:

- **valid_extensions**: Maps MIME content types to suffix labels (e.g., `image/jpeg: "[jpg]"`)
- **passive_extensions**: Maps file extensions to suffix labels (e.g., `".jpg": "[jpg]"`)

### Location

The `extensions.yaml` file is stored in `~/.config/linkinspector/extensions.yaml`. 

**Automatic Setup:**
- The directory `~/.config/linkinspector` is automatically created if it doesn't exist
- The `extensions.yaml` file is automatically downloaded from GitHub if it doesn't exist
- The file is downloaded from: `https://raw.githubusercontent.com/rix4uni/linkinspector/refs/heads/main/extensions.yaml`

### Customization

You can customize the `extensions.yaml` file in `~/.config/linkinspector/extensions.yaml` to add or modify extension mappings according to your needs. The file uses standard YAML syntax. Your customizations will persist across tool updates.

## Usage
```console
linkinspector is a command-line tool that analyzes URLs to retrieve HTTP status codes, content lengths, and content types.

Usage:
  linkinspector [flags]

Flags:
INPUT:
   -u, -target string  Single URL to check
   -l, -list string    File containing list of URLs to check

PROBES:
   -passive  Enable passive mode to skip requests for specific extensions

MATCHERS:
   -mc, -match-code string    Match response with specified status code (e.g., -mc 200,302)
   -ml, -match-length string  Match response with specified content length (e.g., -ml 100,102)
   -mt, -match-type string    Match response with specified content type (e.g., -mt "application/octet-stream,text/html")
   -ms, -match-suffix string  Match response with specified suffix name (e.g., -ms "ZIP,PHP,7Z")

FILTERS:
   -fc, -filter-code string    Filter response with specified status code (e.g., -fc 403,401)
   -fl, -filter-length string  Filter response with specified content length (e.g., -fl 23,33)
   -ft, -filter-type string    Filter response with specified content type (e.g., -ft "text/html,image/jpeg")
   -fs, -filter-suffix string  Filter response with specified suffix name (e.g., -fs "CSS,Plain Text,html")

OUTPUT:
   -o, -output string  File to write output results
   -json               Output in JSON format
   -json-type string   Output in JSON type, MarshalIndent or Marshal (default "MarshalIndent")

RATE-LIMIT:
   -t, -threads int  Number of threads to use (default 50)

CONFIGURATIONS:
   -H string  Custom User-Agent header for HTTP requests (default "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36")

DEBUG:
   -verbose        Enable verbose output for debugging purposes
   -version        Print the version of the tool and exit
   -silent         silent mode
   -nc, -no-color  disable colors in cli output

OPTIMIZATIONS:
   -timeout int  HTTP request timeout duration (in seconds) (default 30)
   -insecure     Disable TLS certificate verification
   -delay value  Duration between each HTTP request (e.g., 200ms, 1s) (default -1ns)
```

## Matchers vs Filters

**Matchers** are used to **include** responses that match specific criteria:
- If a matcher is specified, only responses matching the criteria will be shown
- Multiple values can be specified separated by commas
- If no matcher is specified, all responses are shown

**Filters** are used to **exclude** responses that match specific criteria:
- If a filter is specified, responses matching the filter criteria will be excluded
- Multiple values can be specified separated by commas
- Filters are applied after matchers

**Example**: Using both matchers and filters together:
- `-mc 200,302` (match only 200 and 302 status codes)
- `-fc 404` (filter out 404 status codes)
- Result: Shows only 200 and 302 responses, excluding any 404s

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

#### Using Matchers
```bash
# Match only 200 and 302 status codes
└─# echo "https://example.com" | linkinspector -mc 200,302

# Match specific content types
└─# cat urls.txt | linkinspector -mt "application/pdf,application/zip"

# Match specific suffixes
└─# cat urls.txt | linkinspector -ms "ZIP,PDF,PHP"
```

#### Using Filters
```bash
# Filter out 403 and 401 status codes
└─# cat urls.txt | linkinspector -fc 403,401

# Filter out specific content types
└─# cat urls.txt | linkinspector -ft "text/html,image/jpeg"

# Filter out specific suffixes
└─# cat urls.txt | linkinspector -fs "CSS,Plain Text,html"
```

#### Combining Matchers and Filters
```bash
# Match 200 status codes but filter out HTML content
└─# cat urls.txt | linkinspector -mc 200 -ft "text/html"

# Match ZIP files but filter out small files
└─# cat urls.txt | linkinspector -ms "ZIP" -fl 0,100
```

#### Passive Mode
```bash
# Use passive mode to check extensions without making requests
└─# cat urls.txt | linkinspector -passive
```

#### JSON Output
```bash
# Output results in JSON format
└─# cat urls.txt | linkinspector -json

# Output in compact JSON format
└─# cat urls.txt | linkinspector -json -json-type Marshal
```

## Supported types

linkinspector supports a wide variety of file types including images, videos, audio files, archives, documents, fonts, programming languages, and more. The complete list of supported types and their MIME type mappings can be found in the `extensions.yaml` file located at `~/.config/linkinspector/extensions.yaml`.

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

## Extension Sources
- https://gist.github.com/ppisarczyk/43962d06686722d26d176fad46879d41
- https://github.com/github-linguist/linguist/blob/main/lib/linguist/languages.yml
