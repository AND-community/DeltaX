# and-plugin-DeltaX — DeltaX

**English** · [Türkçe](README.tr.md)

Set up private groups for [AND](https://github.com/AND-community/AND) and develop projects with their
own version control system. Code and project history travel only between the members who make up the
group, completely outside the central forum/chat network, end-to-end encrypted — the group's **own
private network**.

---

## Overview

`and-plugin-DeltaX` has three layers:

1. **Groups** — private communities of members invited by the founder, sharing a randomly generated AES-256 secret (`Sir`).
2. **Projects** — inside every group, a content-addressed (blob/tree/commit based) version control repository (with branches, commits, log, diff and three-way merge).
3. **Synchronisation** — over AND's DM (`/and/dm/1.0.0`) and file transfer (`/and/file/2.0.0`) protocols, strictly peer-to-peer, with packets encrypted with the group secret. No data ever touches the forum, the GossipSub chat channels or any other central component.

At startup the plugin also reads AND's active theme with `GET /api/v1/theme` (e.g. the "Pembiş"
theme selected with the **pembis** plugin) and sets its own colours to match — so it opens in the
same colour scheme as the main application.

---

## Installation

DeltaX lives in its own repository; it is cloned into the `Eklentiler/` folder of the
[AND](https://github.com/AND-community/AND) repository and built from AND's root directory:

```bash
git clone https://github.com/AND-community/DeltaX.git Eklentiler/DeltaX
go build -o Eklentiler/DeltaX/and-plugin-DeltaX ./Eklentiler/DeltaX

# Windows
go build -o Eklentiler/DeltaX/and-plugin-DeltaX.exe ./Eklentiler/DeltaX
```

AND discovers `Eklentiler/DeltaX/plugin.json` next to its own binary together with the binary above.

---

## Usage

Choose **DeltaX** in AND's main menu. The plugin opens in AND's interface language (Turkish,
English, German, Spanish, French or Russian).

### Appearance

DeltaX uses the same visual language as AND itself: a full-screen frame, a breadcrumb at the top of
every screen showing where you are (`DeltaX › Group › Project › Log`), on the right the active branch
(`⎇ ana` — "ana" is the default branch name — with `●` if there are uncommitted changes) or the number
of members online, lists with a selection bar, and the screen's shortcuts at the bottom. Long lists
show as much as fits on the screen ("↑ 3 more"). The colours come from the theme selected in AND.

### Main menu

| Key | Action |
|-----|--------|
| `↑/↓`, `j/k` | Move |
| `enter` | Select |
| `q`/`esc` | Quit |

Options: **My groups**, **Create New Group**, **Invitations**, **Identity Details**, **Quit**.
On a wide terminal a summary sits on the right: the number of groups, your group mates who are
online, pending invitations and the **latest commits** across all projects.

### Shortcut help, progress indicator and confirmation dialogs

- **`?` — shortcuts**: on most list/menu screens (except screens with text input), pressing `?` opens all of that screen's shortcuts in a large, readable box; any key closes it.
- **Real progress indicator**: operations that run in the background, such as Commit/Push/Pull/Merge/Export, are shown with a spinning indicator instead of static text — you can see that the operation hasn't frozen.
- **Confirmation dialogs**: like leaving or closing a group, **switching branches** and **merging** also use the same dialog that asks for confirmation with `y`/`n` — both delete and rewrite **everything in the working folder, including untracked files and files ignored with `.projeagignore`**, and the dialog says so explicitly. (Exception: deleting a single file on the Files screen still uses the quick "press d twice" pattern — a modal dialog would only slow down a low-risk, frequent operation.)
- **Search in My groups**: `/` filters live by group name or founder name (the same pattern as the search on the Log screen); `esc` leaves the search.

### Group flow

1. **Create New Group** — enter the group name; a random group key is generated.
2. In the group panel, invite a member by Peer ID with `i` (the founder or an admin can invite). The screen suggests the **AND nodes** that are connected right now and not yet in the group; selecting one with `↑/↓` writes its Peer ID into the box. If the person is offline, AND keeps the invitation in its outbox and delivers it when they connect (for up to 7 days).
3. The invited person accepts with `a` (or rejects with `r`) on the **Invitations** screen — the answer is sent back to the inviter automatically and the current member list is re-broadcast to everyone in the group, so everyone's list converges. Even if the inviter has closed DeltaX at that moment, the answer isn't lost: it is processed on the next start.
4. Create a new project in the group panel with `n`.

### Appoint an admin (backup founder)

In the group panel the founder can make a member an **admin** with `y` (or remove the role). Admins
can invite members and close the group just like the founder — so the group isn't left without
anyone in charge if the founder loses their device or access. Only the real founder can appoint or
remove admins; an admin can't pass this right on.

### Leaving / closing a group

Press `x` in the group panel, then confirm with `y`:

- **If you are a regular member** → *Leave Group*: the founder and all admins are notified (whoever is reachable removes you from the list and re-broadcasts the current list); the group's local record is deleted on your side.
- **If you are the founder or an admin** → *Close Group*: all members are notified that the group was dissolved, and on the receiving side the group disappears from their lists automatically; the group's local record is deleted for everyone.

In both cases **your project files stay on disk** — no local work is deleted, only the group/roster
itself is removed. `n` cancels.

### Project panel

The top of the panel shows the project's status: the active branch, the last commit (message,
author, when), changes in the working folder (`~2 changed  +1 new`), members online and remote tips
that carry unmerged commits. Every action has a shortcut that works directly, without selecting it
in the list.

| Key | Action | Description |
|-----|-------|----------|
| `d` | Changes | Uncommitted files; `enter` shows the diff, `g` reverts the change (the file goes back to its state in the last commit, a new file is deleted; asks for confirmation), `c` commits |
| `c` | Commit | Take a snapshot, write a message and create a new commit; the screen shows what will be committed |
| `l` | Log | Commits on the active branch; `/` searches by message/author/hash, `enter` shows the **commit details** |
| `b` | Branches | Lists the branches with their last commits; `enter` switches to a branch, `n` creates a branch, `d` deletes a branch (the active branch can't be deleted; its commits stay in the history) |
| `p` | Push | Send new commits as an encrypted packet to one member or (`a`) to everyone online; the last commit each member knows is shown |
| `m` | Remote Branches (Merge) | Tips received from members, with **how many new commits** each has or whether it is "up to date"; `enter` merges three-way |
| `u` | Request Pull from Member | Ask one member or (`a`) everyone online to send their changes |
| `f` | Files (Edit) | Create/edit/delete files without leaving AND (built-in text editor) |
| `o` | Open Working Folder | Open the folder with the project's files in the system file manager |
| `e` | Export (.zip) | Pack the project's whole history (objects + working folder) into a single .zip |

**Commit details** (`enter` in the Log): the commit's full ID, author, date, its parent commits (two
for a merge commit), the message and the files it changed. `enter` on a file shows what changed in it
in that commit; `b` creates a new branch from this commit (to go back to an older version and
continue from there).

**Diffs** of long files show only the changed places and three lines around them; the unchanged parts
in between are folded as "⋯ 42 lines unchanged". The numbers of added and deleted lines are shown at
the top.

The project's files are kept in the **Working Folder** (its path is shown in the project panel) —
edit the files there with your own editor or with the plugin's **Files (Edit)** screen, then create a
commit with **Commit**.

### .projeagignore

By putting a `.projeagignore` file (one pattern per line) in the root of the working folder you can
keep specific files/folders out of snapshots (Status/Commit/Push) — e.g. build output or
`node_modules`. A pattern without `/` (e.g. `node_modules`, `*.log`) excludes a path (and everything
under it) if it matches any segment of the path; patterns that contain `/` match the full path from
the root. The file itself is tracked normally, so the rules are shared among the group members too.

### Merge conflicts

If the same file was changed differently on both sides, it is written to the working folder with these
markers (`YEREL` means local, `UZAK` means remote):

```
<<<<<<< YEREL
(your version)
=======
(the remote version)
>>>>>>> UZAK
```

Edit the file by hand, remove the markers and then complete the merge commit with **Commit**
(`MERGE_HEAD` is cleared automatically).

---

## Technical details

### Version control model

A content-addressed version control model: `Blob` (raw content), `Agac` (tree: path → blob hash map)
and `Commit` (tree + parent(s) + author/message/time). Objects are addressed by
`sha256(type + content)` and stored deduplicated under `nesneler/<hash[:2]>/<hash[2:]>`. Switching
branches requires a clean working directory; merges find the common ancestor and are applied as a
fast-forward or a three-way merge.

### Network protocol

| Channel | Use |
|-------|----------|
| DM (`/and/dm/1.0.0`) | Small control messages — invitation, member/admin list update, push notice, pull request, leaving a group, closing a group. They travel as base64 JSON prefixed with `AND-PROJEAG/1\|` (prefix: `protokol.DeltaXOnEki`); AND doesn't count messages with this prefix as unread, and Özel Chat doesn't show them. Messages that change the group state (invitation and its answer, member list, leaving, closing) go through the outbox; the push notice and the pull request go directly, because the file transfer needs both sides to be online. Incoming messages are read from AND's DM history (see Persistence). |
| File transfer (`/and/file/2.0.0`) | The actual data — a gzipped tar archive of commit/tree/blob objects named `projeag-<groupID>-<projectID>-<time>.pack`. Before sending, the archive is encrypted with the group secret using AES-256-GCM. |

### Security model

- **Group secret:** every group has a random 32-byte AES key that is sent by DM only to invited members. Packets are encrypted with this key — nobody outside the group (even if connected to the same AND network) can decrypt the content.
- **Integrity:** the name of every object in a packet is its own sha256 hash; when unpacking, the hash is recomputed and checked, and entries that don't match are silently skipped.
- **Automatic file consent only from group members:** AND's file-transfer consent flow is preserved. A file with the `projeag-` prefix is accepted automatically only if the sender's Peer ID as verified by libp2p (`FileConsentReq.SenderID` — it can't be spoofed) belongs to a member of a group known locally; otherwise manual consent is requested as usual.
- **Target check on pull requests:** when a member's pull request is answered automatically, the Peer ID given in the request is compared with the group member list known locally at that moment — so nobody can redirect data outside the group by giving a fake target Peer ID.
- **Founder and admin verification.** Invitation/roster/close messages are accepted only if the sender's peer ID matches the already known `KurucuPeer` (founder) or `Yoneticiler` (admins) list — someone who has never been a member of the group can't take over the roster with a fake message.
- **Only invited people can join.** An "accept" answer is applied only if an invitation for that group was really sent to that peer and the answer arrived within 30 days. Sent invitations are kept on disk; an invitation counts as used once it has been answered (accepted or rejected).
- **Old requests are not answered.** No packet is sent for a pull request older than one hour that arrived while DeltaX was closed; an invitation older than 30 days is not listed.

### Persistence

All data is stored locally under `AND_DATA_DIR/DeltaX/` — nothing is sent to a central server:

```
DeltaX/
  gruplar/<groupID>.json
  projeler/<projectID>/meta.json       (branches, active branch, remote branches)
  projeler/<projectID>/nesneler/…      (content-addressed commit/tree/blob store)
  projeler/<projectID>/calisma/…       (working folder — this is where you edit the files)
  projeler/<projectID>/MERGE_HEAD      (exists only while a conflicting merge is pending)
  bekleyen_davetler.json               (invitations you received and haven't answered yet)
  gonderilen_davetler.json             (invitations you sent that are waiting for an answer)
  dm_imleci.json                       (how far AND's DM history has been processed)
```

DeltaX reads group messages from AND's DM history and writes how far it has processed them to
`dm_imleci.json`: messages that arrived while it was closed are processed in order on the next
start, and the same message is never processed twice. On the very first start earlier messages are
not processed. With an older AND that has no history, only messages that arrive while DeltaX is open
are received.

---

## Limitations

- **File transfer needs both sides to be online.** Group messages (invitation, member list, leaving, closing) are delivered to an offline recipient when they connect, and those that arrive while DeltaX is closed aren't lost; but a push packet and a pull request only work while the other side is connected at that moment.
- **AND's DM history is limited.** The history keeps at most 300 machine messages (separately from people's messages); if DeltaX stays closed for a very long time, the oldest group messages may drop out. The member list converges again with the next update.
- **The founder can't be changed or handed over.** Only an "admin" (backup founder) can be appointed — see Usage/"Appoint an admin". The founder can't "leave" a group — they can only close the group entirely.
- **Simplified merging.** A text-based, line-level three-way merge is performed; for binary files a conflict always marks the whole file.
- **Large files.** There are limits of 200 MB for single files included in a snapshot, 512 KB for line-based diffs and 2 MB for files that can be opened in the built-in editor; anything larger is skipped/reported (an external editor/file manager can be used).
- **No distinction between tracked and untracked files with `.projeagignore`.** If a file was committed earlier and later added to the ignore list, it shows up as "deleted" on the next Status/Commit screens.

---

## Manifest

```json
{
  "name":        "DeltaX",
  "label":       "DeltaX",
  "version":     "1.7.1",
  "description": "Özel gruplar kurup kendi sürüm kontrol sistemiyle birlikte proje geliştirme — merkezi ağ dışında, yalnızca grup üyeleri arasında uçtan uca şifreli senkronizasyon",
  "author":      "AND",
  "permissions": ["dm", "file"],
  "requires":    []
}
```

It asks for the `dm` (group messages) and `file` (project packets) permissions the first time it is
opened. (The description in the manifest is in Turkish; AND shows it translated in the other
interface languages.)

---

## Source

Source code: [this repository](.)

| File | Contents |
|-------|--------|
| `main.go` | Manifest, entry point, styles |
| `model.go` | Bubble Tea model, async commands, DM envelope handling (`zarfIsle`) |
| `isleyiciler.go` | `Update`: one handler per message type; refreshes a screen's summary when you enter it |
| `gecmis.go` | Reading from the DM history, the cursor, the ledger of sent invitations |
| `istemci.go` | The AND API used by DeltaX (tests use a fake) |
| `keys.go` | Keyboard event handlers per screen |
| `view.go` | Screen rendering (AND design system: breadcrumb, lists, two-column panels, dialogs) |
| `inceleme.go` | Files changed by a commit and their diffs, reverting a change, the number of new commits on remote tips, diff folding, relative time |
| `store.go` | Group/project persistence |
| `vcs.go` | Content-addressed object store, branch/commit/diff/merge, log search |
| `crypto.go` | AES-256-GCM encryption with the group secret |
| `sync.go` | DM control protocol, packet creation/application, push/pull/invite/admin |
| `ignore.go` | `.projeagignore` pattern matching |
| `editor.go` | Built-in file listing/reading/writing/deleting |
| `export.go` | Exporting a project as a `.zip` |
| `ceviri.go` | Embeds the translations of the plugin's texts (`ceviri.json`); the interface opens in AND's language |

---

## Version history

| Version | Change |
|-------|------------|
| 1.7.1 | State files and the working folder are written atomically: a crash or power cut while saving can't corrupt group, project and object files |
| 1.7.0 | Interface in AND's language (6 languages) |
| 1.6.0 | New look: AND design system (breadcrumb, active branch, lists with a selection bar and windowing, summary and latest commits on the main menu, projects and members side by side in the group panel, status card and shortcut actions in the project panel). Commit details and per-commit file diffs, creating a branch from a commit, reverting a change, deleting a branch, the number of unmerged commits on remote tips, requesting a pull from everyone online, suggestions of connected AND nodes when inviting, folded diffs. Fixed a fake blank line in the diff of a new or deleted file |
| 1.5.0 | Group messages are never missed: incoming ones are read from AND's DM history (those that arrived while it was closed are processed at startup), those that change the group state go through the outbox; sent invitations are kept on disk (30 days), and rejections are reported; old pull requests are not answered; when inviting, the Peer ID format, inviting yourself and inviting an existing member are checked; machine messages don't show up in Özel Chat or in AND's unread count |
| 1.3.0 | The plugin was renamed from **ANDSnap** to **DeltaX** (including the directory, the manifest and the data folder `AND_DATA_DIR/DeltaX/`) |
| 1.2.0 | The plugin was renamed from **proje_ag** to **ANDSnap** (including the directory, the manifest and the data folder `AND_DATA_DIR/ANDSnap/`) |
| 1.1.0 | `.projeagignore`, appointing an admin (backup founder), built-in file editor, search in the log, exporting a project as a `.zip`, applying the AND theme automatically, quitting the main menu with `esc` |
| 1.0.0 | First version — groups, its own version control system, end-to-end encrypted P2P push/pull/merge |
