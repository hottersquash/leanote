#!/bin/bash
set -e
LEANOTE=/home/byan/leanote
SCRIPTS=/tmp/leanote-anchor-fix

mkdir -p "$SCRIPTS"
cp "$0" "$SCRIPTS/" 2>/dev/null || true

# 1. Deploy JS fixes
cp note-anchor-fix.js "$LEANOTE/public/js/note-anchor-fix.js"
cp blog-anchor-fix.js "$LEANOTE/public/blog/js/blog-anchor-fix.js"

# 2. Fix md2html slugify (Unicode property escapes need "u" flag)
python3 << 'PY'
from pathlib import Path
p = Path("/home/byan/leanote/public/libs/md2html/md2html.js")
text = p.read_text(encoding="utf-8")
old = "var nonWordChars = new RegExp('[^\\\\p{L}\\\\p{N}-]', 'g');"
new = "var nonWordChars = new RegExp('[^\\\\p{L}\\\\p{N}-]', 'gu');"
if old in text:
    text = text.replace(old, new, 1)
    p.write_text(text, encoding="utf-8")
    print("patched md2html slugify")
else:
    print("md2html slugify already patched or pattern changed")
PY

# 3. Patch note.html to load note-anchor-fix.js
NOTE_HTML="$LEANOTE/app/views/note/note.html"
if ! grep -q note-anchor-fix.js "$NOTE_HTML"; then
  sed -i 's|<script src="/js/markdown-v2.min.js"></script>|<script src="/js/markdown-v2.min.js"></script>\n<script src="/js/note-anchor-fix.js"></script>|' "$NOTE_HTML"
  echo "patched note.html"
fi

NOTE_DEV="$LEANOTE/app/views/note/note-dev.html"
if [ -f "$NOTE_DEV" ] && ! grep -q note-anchor-fix.js "$NOTE_DEV"; then
  sed -i 's|<script src="/js/markdown-v2.min.js"></script>|<script src="/js/markdown-v2.min.js"></script>\n<script src="/js/note-anchor-fix.js"></script>|' "$NOTE_DEV"
  echo "patched note-dev.html"
fi

# 4. Patch blog footer to load blog-anchor-fix.js on all blog pages
for theme in default elegant nav_fixed; do
  FOOTER="$LEANOTE/public/blog/themes/$theme/footer.html"
  if [ -f "$FOOTER" ] && ! grep -q blog-anchor-fix.js "$FOOTER"; then
    sed -i 's|<script src="{{\$.bootstrapJsUrl}}"></script>|<script src="{{\$.bootstrapJsUrl}}"></script>\n<script src="{{\$.siteUrl}}/public/blog/js/blog-anchor-fix.js"></script>|' "$FOOTER"
    echo "patched $theme/footer.html"
  fi
done

# 5. Patch blog post.html to init anchors after md2Html
for theme in default elegant nav_fixed; do
  POST="$LEANOTE/public/blog/themes/$theme/post.html"
  if [ -f "$POST" ] && ! grep -q initBlogHeadingAnchors "$POST"; then
    sed -i 's|initNav();|initNav();\n    if (window.initBlogHeadingAnchors) initBlogHeadingAnchors("#content");|' "$POST"
    echo "patched $theme/post.html"
  fi
done

# 6. Fix common.js scrollTo shadowing native window.scrollTo
COMMON="$LEANOTE/public/blog/js/common.js"
python3 << 'PY'
from pathlib import Path
p = Path("/home/byan/leanote/public/blog/js/common.js")
text = p.read_text(encoding="utf-8")
text = text.replace("function scrollTo(self, tagName, text)", "function scrollToHeading(self, tagName, text)")
text = text.replace('onclick="window.scrollTo(this,', 'onclick="scrollToHeading(this,')
if "function scrollToHeading" in text:
    p.write_text(text, encoding="utf-8")
    print("patched common.js scrollToHeading")
PY

echo "Done. Restart leanote if needed."
