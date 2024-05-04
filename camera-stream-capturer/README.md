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

```bash
brew install gcc
brew install pkg-config
brew install libav
brew install ffmpeg
```

## References

- [Kerberos.io agent](https://github.com/kerberos-io/agent)
- [https://github.com/go101/go101/wiki/CGO-Environment-Setup](https://github.com/go101/go101/wiki/CGO-Environment-Setup)
- [https://medium.com/@mfkhao2009/set-up-ffmpeg-development-enviroment-on-macos-2523f7d3b2e8](https://medium.com/@mfkhao2009/set-up-ffmpeg-development-enviroment-on-macos-2523f7d3b2e8)
- [https://cloud.google.com/vision?hl=en#demo](https://cloud.google.com/vision?hl=en#demo)
