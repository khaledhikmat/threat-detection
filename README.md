This project is an experimentation to create a simplified threat detection solution.

```bash
go work init ./shared ./camera-stream-capturer 
```

To add new modules:

```bash
go work edit -use ./ccure-notifier 
```

To drop modules:

```bash
go work edit -dropuse ./ccure-notifier 
```
