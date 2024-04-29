Threat detection experimentation to leverage [https://github.com/bluenviron/gortsplib](https://github.com/bluenviron/gortsplib) to create a simplified camera stream capturer that continuously chunks up frames and generates MP4 video clips.

The MP4 video clips can then be stored on a Cloud storage such as Amazon S3. Each clip storage can trigger additional processing (cognitive or behavioral recognition) and possibly generate alerts.

```bash
go mod init github.com/khaledhikmat/threat-detection/camera-stream-capturer
go get -u github.com/joho/godotenv
go get -u github.com/google/uuid
go get -u github.com/kerberos-io/agent/machinery
```

There are some additional dependencies on `C` bindings and libraries:

```bash
brew install gcc
brew install pkg-config
brew install libav
brew install ffmpeg
```

## Enhancements

- Every 5 seconds, emit a heartbeat signal that updates a key/value store to indicate whether the capturer is alive.
- Upon startup, reach out to SOICAT to determine the cameras that are not being captured or the capturer expired (based on heartbeat).
- Support multiple cameras.
- Support random MP4 generation.
- Support Dockerfile with dependencies. 
- Support configuration:
    - Maximum camera streams
- Support SOICAT Enhancement to allow for camera/device capture attributes:
    - Is Capture required?
    - RTSP URL
    - others

- Queue to Go Pipeline
- Try to not need the Kerberos agent
- Abstract the 
## References

- [Kerberos.io agent](https://github.com/kerberos-io/agent)
- [https://github.com/go101/go101/wiki/CGO-Environment-Setup](https://github.com/go101/go101/wiki/CGO-Environment-Setup)
- [https://medium.com/@mfkhao2009/set-up-ffmpeg-development-enviroment-on-macos-2523f7d3b2e8](https://medium.com/@mfkhao2009/set-up-ffmpeg-development-enviroment-on-macos-2523f7d3b2e8)
- [https://cloud.google.com/vision?hl=en#demo](https://cloud.google.com/vision?hl=en#demo)
