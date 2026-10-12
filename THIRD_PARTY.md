# Third-party software

| Name | Version | License | Shipped in the app | Notice | Credited in About |
|---|---|---|---|---|---|
| github.com/egoist/mygo | v0.3.5 | MIT | yes | `resources/ThirdPartyLicenses/mygo/LICENSE` | no |
| github.com/ebitengine/purego | v0.11.1 | Apache-2.0 | yes | `resources/ThirdPartyLicenses/purego/LICENSE` | no |
| github.com/go-text/typesetting | v0.3.5 | BSD-3-Clause | yes | `resources/ThirdPartyLicenses/typesetting/LICENSE` | no |
| golang.org/x/image | v0.46.0 | BSD-3-Clause | yes | `resources/ThirdPartyLicenses/image/LICENSE` | no |
| golang.org/x/text | v0.43.0 | BSD-3-Clause | yes | `resources/ThirdPartyLicenses/text/LICENSE` | no |

## Notes

The MyGo CLI is a module-pinned build tool in the same module version; it does not ship.
Transitive modules are listed because their code ships. Go standard-library components
retain their Go license; a copy ships under resources/ThirdPartyLicenses/go/.

## Documentation

`docs/design.md` and `design/` summarize, in our own words, the UI documentation of MyGo v0.3.5 (MIT License,
Copyright (c) egoist): <https://github.com/egoist/mygo/tree/v0.3.5/docs/ui>. No upstream text or code is shipped in
the app.
