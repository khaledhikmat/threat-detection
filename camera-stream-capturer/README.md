Threat detection experimentation to leverage [https://github.com/bluenviron/gortsplib](https://github.com/bluenviron/gortsplib) to create a simplified camera stream capturer that continuously chunks up frames and generates MP4 video clips.

The MP4 video clips can then be stored on a Cloud storage such as Amazon S3. Each clip storage can trigger additional processing (cognitive or behavioral recognition) and possibly generate alerts.

Please note that I borrowed some code from: [https://github.com/kerberos-io/agent/machinery](https://github.com/kerberos-io/agent/machinery)

```bash
go mod init github.com/khaledhikmat/threat-detection/camera-stream-capturer
go get -u github.com/joho/godotenv
go get -u github.com/google/uuid
go get -u github.com/dapr/go-sdk
go get -u github.com/mitchellh/mapstructure

```

There are some additional dependencies on `C` bindings and libraries:

## MacOS

```bash
brew install gcc
brew install pkg-config
brew install libav
brew install ffmpeg
```

## Ubuntu

```bash
sudo apt update
sudo apt apt-file
sudo apt install build-essential
gcc --version
sudo apt install ffmpeg
ffmpeg -version
dpkg -L ffmpeg # to find out where it installed the libs
apt-file update
apt-file search libavcodec.pc
export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:/usr/lib/x86_64-linux-gnu/pkconfig
pkg-config --cflags -- libavcodec libavutil libswscale
# https://github.com/opencv/opencv/issues/5930
.pc files must be created manually using vim
sudo apt-get install libavcodec-dev #asked copilot
sudo apt-get install libswscale-dev #asked copilot
```

## References

- [Kerberos.io agent](https://github.com/kerberos-io/agent)
- [https://github.com/go101/go101/wiki/CGO-Environment-Setup](https://github.com/go101/go101/wiki/CGO-Environment-Setup)
- [https://medium.com/@mfkhao2009/set-up-ffmpeg-development-enviroment-on-macos-2523f7d3b2e8](https://medium.com/@mfkhao2009/set-up-ffmpeg-development-enviroment-on-macos-2523f7d3b2e8)
- [https://cloud.google.com/vision?hl=en#demo](https://cloud.google.com/vision?hl=en#demo)
