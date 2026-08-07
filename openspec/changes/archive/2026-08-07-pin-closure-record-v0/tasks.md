## 1. Record what a reviewer last looked at

- [x] 1.1 Add the closure record — package count, install-script count, digest — to the release source lock and its contract type
- [x] 1.2 Add the upstream publication and adoption dates to the pinned runtime and compiler
- [x] 1.3 Populate the record from this repository's actual pinned set and the registries' publication dates

## 2. Refuse a closure nobody recorded

- [x] 2.1 Compare the computed closure against the record where release inputs are loaded, so building, verifying and inspecting all inherit it
- [x] 2.2 Name which of the three facts disagreed in the refusal
- [x] 2.3 Require both adoption dates and refuse an adoption preceding its publication

## 3. Version the expanded document

- [x] 3.1 Move the schema identifier to `goalrail.setup-source-lock/v2`
- [x] 3.2 Register v2 in the operator contract registry, describing what it adds and that the document is a build input rather than a published asset

## 4. State the mechanism and its limit

- [x] 4.1 Describe the record and the check in the repository contract
- [x] 4.2 State that the dates are disclosure rather than a gate, and that the freshness rule is not decided here

## 5. Prove it

- [x] 5.1 Table-driven cases over the three closure facts, against a copy of this repository's real pinned set
- [x] 5.2 Cases for a missing date and an adoption preceding publication
- [x] 5.3 A case asserting the shared input path refuses, so build and verify cannot bypass the check
- [x] 5.4 A case asserting the expanded shape under the previous identifier is refused
- [x] 5.5 `go vet ./...` and `go test ./...`
