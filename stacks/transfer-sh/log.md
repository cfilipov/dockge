# transfer.sh Stack

## Source
https://github.com/dutchcoders/transfer.sh

## Description
Easy and fast file sharing from the command-line. Supports local/S3/GDrive/Storj storage, max-downloads limits, max-days TTL, server-side encryption, and VirusTotal scanning.

## Services
- **transfer-sh** - Main application (dutchcoders/transfer.sh:latest) on port 8080

## Reference
Uses local storage provider. Upload via `curl --upload-file ./file.txt http://localhost:8080/file.txt`.
