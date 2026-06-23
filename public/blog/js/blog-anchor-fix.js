/**
 * Fix heading anchor navigation for blog markdown content (abstracts and posts).
 */
(function($) {
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
        $(container).find('h1,h2,h3,h4,h5,h6').each(function() {
            var $h = $(this);
            if ($h.attr('id')) {
                used[$h.attr('id')] = true;
                return;
            }
            var id = slugify($h.text()) || 'title';
            var anchor = id;
            var n = 0;
            while (used[anchor]) {
                anchor = id + '-' + (++n);
            }
            used[anchor] = true;
            $h.attr('id', anchor);
        });
    }

    function scrollToHash(container, hash) {
        if (!hash || hash === '#') {
            return false;
        }
        ensureHeadingIds(container);
        var id = decodeURIComponent(hash.replace(/^#/, ''));
        var target = document.getElementById(id);
        if (!target) {
            return false;
        }
        var top = $(target).offset().top - 60;
        $('html, body').animate({ scrollTop: top }, 200);
        return true;
    }

    function initContainer(container) {
        ensureHeadingIds(container);
        $(container).off('click.blogAnchorFix', 'a[href^="#"]');
        $(container).on('click.blogAnchorFix', 'a[href^="#"]', function(evt) {
            var hash = this.getAttribute('href');
            if (!hash || hash === '#') {
                return;
            }
            evt.preventDefault();
            scrollToHash(container, hash);
        });
    }

    window.initBlogHeadingAnchors = function(selector) {
        $(selector || '.desc, #content').each(function() {
            initContainer(this);
        });
    };

    $(function() {
        initBlogHeadingAnchors('.each-post .desc, #content');
        if (location.hash) {
            setTimeout(function() {
                scrollToHash(document, location.hash);
            }, 300);
        }
    });
})(jQuery);
