/**
 * Fix in-note heading anchor navigation in Markdown preview.
 * Handles links in preview body and the right-side Note nav (#leanoteNavContentMd).
 */
(function() {
    function slugify(text) {
        return (text || '').toLowerCase()
            .replace(/\s/g, '-')
            .replace(/[^\w\u4e00-\u9fff\-]+/g, '')
            .replace(/-+/g, '-')
            .replace(/^-+|-+$/g, '');
    }

    function ensureHeadingIds(container) {
        if (!container) {
            return;
        }
        var used = {};
        var headings = container.querySelectorAll('h1, h2, h3, h4, h5, h6');
        for (var i = 0; i < headings.length; i++) {
            var h = headings[i];
            if (h.id) {
                used[h.id] = true;
                continue;
            }
            var id = slugify(h.textContent) || 'title';
            var anchor = id;
            var n = 0;
            while (used[anchor]) {
                anchor = id + '-' + (++n);
            }
            used[anchor] = true;
            h.id = anchor;
        }
    }

    function scrollPreviewTo(target) {
        var preview = document.querySelector('.preview-container');
        if (!preview || !target) {
            return;
        }
        var top = target.getBoundingClientRect().top
            - preview.getBoundingClientRect().top
            + preview.scrollTop
            - 8;
        if (preview.scrollTo) {
            preview.scrollTo({ top: top, behavior: 'smooth' });
        } else {
            $(preview).animate({ scrollTop: top }, 200);
        }
    }

    function findAnchorTarget(container, hash) {
        if (!hash || hash === '#') {
            return null;
        }
        var id = decodeURIComponent(hash.replace(/^#/, ''));
        var byId = document.getElementById(id);
        if (byId && container.contains(byId)) {
            return byId;
        }
        var headings = container.querySelectorAll('h1, h2, h3, h4, h5, h6');
        for (var i = 0; i < headings.length; i++) {
            if (headings[i].id === id) {
                return headings[i];
            }
        }
        return null;
    }

    function handleAnchorClick(evt, link) {
        var hash = link.getAttribute('href');
        if (!hash || hash === '#') {
            return false;
        }

        var contents = document.getElementById('preview-contents');
        if (!contents) {
            return false;
        }

        ensureHeadingIds(contents);
        var target = findAnchorTarget(contents, hash);
        if (!target) {
            return false;
        }

        evt.preventDefault();
        evt.stopPropagation();
        evt.stopImmediatePropagation();
        scrollPreviewTo(target);
        return true;
    }

    function bindDelegatedNav(container) {
        if (!container || container.__noteAnchorFixBound) {
            return;
        }
        container.__noteAnchorFixBound = true;
        container.addEventListener('click', function(evt) {
            var link = evt.target.closest('a[href^="#"]');
            if (!link) {
                return;
            }
            handleAnchorClick(evt, link);
        }, true);
    }

    function bindAnchorFix() {
        bindDelegatedNav(document.getElementById('preview-contents'));
        bindDelegatedNav(document.getElementById('leanoteNavContentMd'));
        bindDelegatedNav(document.getElementById('leanoteNavContent'));
        // scrollLink binds the same class inside extension-preview-buttons
        bindDelegatedNav(document.querySelector('.extension-preview-buttons .table-of-contents'));
    }

    function init() {
        bindAnchorFix();
        var contents = document.getElementById('preview-contents');
        if (contents) {
            ensureHeadingIds(contents);
        }
    }

    if (window.eventMgr && eventMgr.addListener) {
        eventMgr.addListener('onReady', init);
        eventMgr.addListener('onPreviewFinished', init);
        eventMgr.addListener('onFileSelected', init);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    window.ensureNoteHeadingIds = ensureHeadingIds;
})();
