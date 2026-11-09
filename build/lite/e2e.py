"""Checks a magpie-lite build against a real Claude Code: is a long message whole?

  python -X utf8 -I build/lite/e2e.py <magpie.exe> <claude.exe> [kb ...]

Each size (8 and 64 KB by default) sends one message with three codes on
Haiku through `magpie.exe claude-tui-helper`, as the gateway runs it, and
compares the user message Claude Code recorded in its transcript with what
was sent, byte for byte. It uses the subscription Claude Code is signed in
with, and removes the sessions it made. Exit status 1 when any differs.
"""
import json
import os
import random
import re
import shutil
import string
import subprocess
import sys
import tempfile
import threading
import time

CLAUDE_DIR = os.environ.get("CLAUDE_CONFIG_DIR") or os.path.expanduser("~/.claude")


def rand(n):
    return "".join(random.choice(string.ascii_uppercase + string.digits) for _ in range(n))


def message(kb):
    tag = "P" + rand(5)
    codes = [f"K-{rand(4)}" for _ in range(3)]
    lines, size = [], 0
    while size < kb * 1024:
        lines.append(f"{tag}L{len(lines):06d} {'probe text filler 中文 ' * 5}\n")
        size += len(lines[-1].encode())
    for code, at in zip(codes, (1, len(lines) // 2, max(len(lines) - 2, 2))):
        lines.insert(at, f"Code {code}.\n")
    text = "".join(lines) + (f"END-{tag}. This message has three codes of the form K-XXXX. "
                             "Reply with exactly the codes you can see, space separated, and nothing else.")
    return tag, codes, text


def unwrap(recorded):
    """Claude Code's record of a long paste: the text between its tags."""
    m = re.fullmatch(r'\n*<pasted_content id="(\w+)">\n(.*)\n</pasted_content id="\1">\n', recorded or "", re.S)
    return m.group(2) if m else recorded


def run(magpie, claude, work, kb):
    tag, codes, text = message(kb)
    env = {k: v for k, v in os.environ.items()
           if not k.upper().startswith(("ANTHROPIC_", "CLAUDE_CODE_")) and k.upper() != "CLAUDECODE"}
    env["MAGPIE_CLAUDE_TUI_TRUST"] = "1"  # as the gateway does in its own folder
    p = subprocess.Popen([magpie, "claude-tui-helper", claude, "-p", "--output-format", "stream-json",
                          "--input-format", "stream-json", "--verbose", "--model", "haiku", "--tools", "",
                          "--strict-mcp-config", "--mcp-config", '{"mcpServers": {}}', "--setting-sources", "",
                          "--dangerously-skip-permissions"],
                         cwd=work, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    err = []
    threading.Thread(target=lambda: err.append(p.stderr.read().decode("utf8", "replace")), daemon=True).start()
    sid, reply = None, ""
    try:
        p.stdin.write((json.dumps({"type": "user", "message": {"role": "user", "content": text}}) + "\n").encode())
        p.stdin.flush()
        t0 = time.time()
        for line in p.stdout:
            try:
                m = json.loads(line)
            except ValueError:
                continue
            if m.get("type") == "system" and m.get("subtype") == "init":
                sid = m.get("session_id")
            if m.get("type") == "assistant":
                reply += "".join(x.get("text", "") for x in m["message"].get("content") or [] if x.get("type") == "text")
            if m.get("type") == "result" or time.time() - t0 > 240:
                break
    except OSError:
        pass
    p.stdin.close()
    try:
        p.wait(30)
    except subprocess.TimeoutExpired:
        p.kill()
    time.sleep(1)
    recorded = None
    for proj in os.listdir(os.path.join(CLAUDE_DIR, "projects")):
        f = os.path.join(CLAUDE_DIR, "projects", proj, f"{sid}.jsonl")
        if sid and os.path.exists(f):
            for line in open(f, encoding="utf8", errors="replace"):
                c = (json.loads(line).get("message") or {}).get("content")
                if isinstance(c, str) and f"END-{tag}" in c:
                    recorded = c
            shutil.rmtree(os.path.dirname(f), ignore_errors=True)  # the work folder's own project
    return {"kb": kb, "sent": len(text.encode()), "whole": unwrap(recorded) == text,
            "codes": sum(c in reply for c in codes), "error": "".join(err).strip()[-300:] or None}


def main():
    if len(sys.argv) < 3:
        sys.exit(__doc__)
    magpie, claude = map(os.path.abspath, sys.argv[1:3])
    work = tempfile.mkdtemp(prefix="magpie-lite-e2e-")
    ok = True
    try:
        for kb in map(int, sys.argv[3:] or ["8", "64"]):
            r = run(magpie, claude, work, kb)
            ok &= r["whole"]
            print(json.dumps(r, ensure_ascii=False), flush=True)
    finally:
        shutil.rmtree(work, ignore_errors=True)
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
