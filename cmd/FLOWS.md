# CLI flows

- Mermaid diagrams of the `gatekeep` command flows.
- All commands except `init` and `gen` share the unlock/save lifecycle below.
- Mutating commands (`init`, `add`, `edit`, `rm`, `passwd`) take an exclusive lock on `<vault>.lock`; `list` and `show` do not.
- `gen` is standalone: it never reads the vault, prompts for a master password, or takes a lock.
- Prompts require a terminal; there is no scripting mode.

## Shared unlock and save

```mermaid
flowchart TD
    A["add / list / show / edit / rm / passwd"] --> B["Load vault from --vault"]
    B --> C{"file exists?"}
    C -->|no| X["error: run gatekeep init first"]
    C -->|yes| D["prompt master password (hidden)"]
    D --> E{"unlock OK?"}
    E -->|no| Y["error: invalid password"]
    E -->|yes| F["vault held in memory"]
    F --> G["run command logic"]
    G --> H{"changed anything?"}
    H -->|yes| I["SaveFile: temp file, fsync, .bak, rename"]
    H -->|no| J["exit"]
```

## `init`

```mermaid
flowchart TD
    A["gatekeep init"] --> L["Acquire &lt;vault&gt;.lock"]
    L --> B{"vault file exists?"}
    B -->|yes| X["error: vault already exists, no overwrite"]
    B -->|no| C["prompt master password (hidden)"]
    C --> D["prompt master password again"]
    D --> E{"match?"}
    E -->|no| Y["error: passwords differ, no file written"]
    E -->|yes| F["New + SaveFile (mode 0600)"]
    F --> G["print vault path"]
```

## `add`

`title`, `username`, and `password` are required. `url` and `notes` are
optional and skipped with an empty line. A field supplied as an argument or
flag is not prompted. The password is always prompted (there is no password
flag).

Before the password prompt, `add` refuses an entry whose title and username
already exist. Titles compare case-insensitively (`gmail` matches `Gmail`);
usernames compare exactly (`ALICE@gmail.com` does not match
`alice@gmail.com`). A matching username under a different title, or a matching
title under a different username, is allowed.

A prompted username that duplicates an entry is warned about on stderr and
asked again, so a typo does not cost the title. A duplicated `--username` flag
is an error instead, because flags are explicit: the error names the existing
entry and suggests `gatekeep edit`. Ctrl-C cancels the re-prompt loop, and
nothing is written until the entry is complete.

```mermaid
flowchart TD
    A["gatekeep add [title]"] --> L["Acquire &lt;vault&gt;.lock"]
    L --> U["Unlock"]
    U --> T{"title argument?"}
    T -->|no| P1["prompt Title (required, re-ask if empty)"]
    T -->|yes| C1{"--username?"}
    P1 --> C1
    C1 -->|yes| D{"same title+username exists?"}
    C1 -->|no| P2["prompt Username (required, re-ask if empty)"]
    P2 --> D
    D -->|no| U3["prompt Password (required, hidden)"]
    D -->|yes, prompted| W["stderr: username already exists"]
    W --> P2
    D -->|yes, from flag| X["error: entry exists + hint (no write)"]
    U3 --> U4{"--url?"}
    U4 -->|yes| U5{"--notes?"}
    U4 -->|no| P3["prompt URL (optional)"]
    P3 --> U5
    U5 -->|yes| S["Add + SaveFile"]
    U5 -->|no| P4["prompt Notes (optional)"]
    P4 --> S
```

## `gen`

`gen` never reads the vault, so it needs no master password and takes no lock.
It prints one password per line on stdout. A policy that cannot be satisfied
fails before anything is printed.

```mermaid
flowchart TD
    A["gatekeep gen"] --> B["parse flags into a policy"]
    B --> C{"policy satisfiable?"}
    C -->|no| X["error: invalid policy, no output"]
    C -->|yes| D["add the required characters"]
    D --> E["fill to length, shuffle with crypto/rand"]
    E --> F["print --count passwords"]
```

## `list`

```mermaid
flowchart TD
    A["gatekeep list [query]"] --> U["Unlock"]
    U --> Q{"query?"}
    Q -->|yes| F["filter by substring of title or username"]
    Q -->|no| G["all entries"]
    F --> S["sort by title"]
    G --> S
    S --> J{"--json?"}
    J -->|yes| K["print JSON (passwords only with --show-passwords)"]
    J -->|no| L["print table: Title, Username, URL"]
```

## Selector resolution (`show`, `edit`, `rm`)

```mermaid
flowchart TD
    A["query"] --> B{"exact title, case-insensitive?"}
    B -->|1 match| S["selected entry"]
    B -->|many matches| P["picker: Title — Username"]
    B -->|no match| C{"substring of title or username?"}
    C -->|1 match| S
    C -->|many matches| P
    C -->|no match| X["error: entry not found"]
    P -->|number| S
    P -->|q or empty| Q["print: Aborted. (exit 0)"]
```

## `show`

The password is masked by default; `--reveal` prints it. The mask has a fixed
length so the real password length does not leak. The hint goes to stderr, so
stdout stays parseable, and an entry without a password is printed unmasked.

```mermaid
flowchart TD
    A["gatekeep show &lt;query&gt;"] --> U["Unlock"]
    U --> R["Resolve selector"]
    R --> P{"--reveal?"}
    P -->|yes| Q["print Title, Username, Password, URL, Notes"]
    P -->|no| M["print Title, Username, ********, URL, Notes"]
    M --> H["stderr: Password hidden; rerun with --reveal"]
```

## `edit`

With at least one field flag, only the provided flags are applied. With no
field flags, an interactive wizard asks for every field and Enter keeps the
current value. Title and username cannot be set empty; clear the optional
fields with `--url=` or `--notes=`.

```mermaid
flowchart TD
    A["gatekeep edit &lt;query&gt;"] --> L["Acquire &lt;vault&gt;.lock"]
    L --> U["Unlock"]
    U --> R["Resolve selector"]
    R --> F{"any field flag?"}
    F -->|no| W["wizard: Title, Username, Password, URL, Notes"]
    W --> W2["Enter keeps current value"]
    W2 --> C{"anything changed?"}
    C -->|no| N["print: no changes"]
    C -->|yes| S["Update + SaveFile"]
    F -->|yes| G["apply only the provided flags"]
    G --> Q{"--password?"}
    Q -->|yes| P["prompt new password (hidden)"]
    Q -->|no| S
    P --> S
```

## `rm`

```mermaid
flowchart TD
    A["gatekeep rm &lt;query&gt;"] --> L["Acquire &lt;vault&gt;.lock"]
    L --> U["Unlock"]
    U --> R["Resolve selector"]
    R --> C{"confirm delete? [y/N]"}
    C -->|y| D["Delete + SaveFile"]
    C -->|other| X["abort, no changes"]
```

## `passwd`

```mermaid
flowchart TD
    A["gatekeep passwd"] --> L["Acquire &lt;vault&gt;.lock"]
    L --> B["Unlock with current master password"]
    B --> C["prompt new master password (hidden)"]
    C --> D["prompt new master password again"]
    D --> E{"match?"}
    E -->|no| X["error, vault unchanged"]
    E -->|yes| F["ChangePassword + SaveFile"]
    F --> K{"--backup?"}
    K -->|no| R["remove vault.gk.bak"]
    K -->|yes| W["warn: backup still uses the old password"]
    R --> G["print: master password changed"]
    W --> G
```
