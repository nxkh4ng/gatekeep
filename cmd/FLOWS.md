# CLI flows

Mermaid diagrams of the `gatekeep` command flows. They describe the intended
M5 behavior: `version`, `init`, `add`, `list`, `show`, `edit`, and `rm` are
implemented; the other commands are wired in the command tree but return
`not implemented yet` until later milestones.

All commands except `init` and `version` share the unlock/save lifecycle
below. Prompts require a terminal; there is no scripting mode.

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
    A["gatekeep init"] --> B{"vault file exists?"}
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

```mermaid
flowchart TD
    A["gatekeep add [title]"] --> U["Unlock"]
    U --> T{"title argument?"}
    T -->|no| P1["prompt Title (required, re-ask if empty)"]
    T -->|yes| C1{"--username?"}
    P1 --> C1
    C1 -->|yes| U3["prompt Password (required, hidden)"]
    C1 -->|no| P2["prompt Username (required, re-ask if empty)"]
    P2 --> U3
    U3 --> U4{"--url?"}
    U4 -->|yes| U5{"--notes?"}
    U4 -->|no| P3["prompt URL (optional)"]
    P3 --> U5
    U5 -->|yes| S["Add + SaveFile"]
    U5 -->|no| P4["prompt Notes (optional)"]
    P4 --> S
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
    J -->|yes| K["print JSON including passwords"]
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
    P --> S
```

## `show`

```mermaid
flowchart TD
    A["gatekeep show &lt;query&gt;"] --> U["Unlock"]
    U --> R["Resolve selector"]
    R --> P["print Title, Username, Password, URL, Notes"]
```

## `edit`

With at least one field flag, only the provided flags are applied. With no
field flags, an interactive wizard asks for every field and Enter keeps the
current value. Title and username cannot be set empty; clear the optional
fields with `--url=` or `--notes=`.

```mermaid
flowchart TD
    A["gatekeep edit &lt;query&gt;"] --> U["Unlock"]
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
    A["gatekeep rm &lt;query&gt;"] --> U["Unlock"]
    U --> R["Resolve selector"]
    R --> C{"confirm delete? [y/N]"}
    C -->|y| D["Delete + SaveFile"]
    C -->|other| X["abort, no changes"]
```

## `passwd`

```mermaid
flowchart TD
    A["gatekeep passwd"] --> B["Unlock with current master password"]
    B --> C["prompt new master password (hidden)"]
    C --> D["prompt new master password again"]
    D --> E{"match?"}
    E -->|no| X["error, vault unchanged"]
    E -->|yes| F["ChangePassword + SaveFile"]
    F --> G["print: master password changed"]
```

## `version`

```mermaid
flowchart TD
    A["gatekeep version"] --> B["print: gatekeep <version>"]
```
