# Changelog

## 1.1.0 - 2026-10-29
- Added: stricter IPv6 validation, including compressed and IPv4-mapped forms
- Changed: network requests now use a 5 second connect timeout and 10 second total timeout
- Changed: `jq` is required only for table output; `-j` and `-i` work without it
- Changed: usage text is sent to stderr when printed after an error
- Fixed: reject an invalid public IP response before using it for the lookup

## 1.0.2 - 2026-08-23
- Added: ISP, organization, and AS details in table output
- Changed: switched the IP lookup endpoint from freeipapi.com to ip-api.com

## 1.0.1 - 2026-08-22
- Changed: IPv4 is now the default; `-6` overrides it

## 1.0.0 - 2026-08-04
- Initial release
