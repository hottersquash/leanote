package service

import (
	"fmt"
	"strings"

	"github.com/leanote/leanote/app/db"
	"github.com/leanote/leanote/app/info"
	"gopkg.in/mgo.v2/bson"
)

func (this *BlogService) normalizeBlogBgImage(image string) string {
	image = strings.TrimSpace(image)
	if image == "" {
		return ""
	}
	if strings.HasPrefix(image, "http") || strings.HasPrefix(image, "/") {
		return image
	}
	return "/" + strings.Trim(image, "/")
}

func (this *BlogService) normalizeBlogBgSize(size string) string {
	switch size {
	case "contain", "auto", "cover", "stretch":
	default:
		size = "cover"
	}
	if size == "stretch" {
		return "100% 100%"
	}
	return size
}

func (this *BlogService) normalizeBlogBgRepeat(repeat string) string {
	switch repeat {
	case "repeat", "repeat-x", "repeat-y", "no-repeat":
		return repeat
	default:
		return "no-repeat"
	}
}

func (this *BlogService) normalizeBlogBgOpacity(opacity int) float64 {
	if opacity <= 0 {
		return 1
	}
	if opacity > 100 {
		opacity = 100
	}
	return float64(opacity) / 100.0
}

func (this *BlogService) normalizeBlogNavPosition(position string) string {
	if position == "right" {
		return "right"
	}
	return "left"
}

func (this *BlogService) fixUserBlogBackground(userBlog *info.UserBlog) {
	if userBlog.HeaderNavBgOpacity <= 0 {
		userBlog.HeaderNavBgOpacity = 100
	}
	if userBlog.PageBgOpacity <= 0 {
		userBlog.PageBgOpacity = 100
	}
	if userBlog.HeaderNavBgSize == "" {
		userBlog.HeaderNavBgSize = "cover"
	}
	if userBlog.PageBgSize == "" {
		userBlog.PageBgSize = "cover"
	}
	if userBlog.HeaderNavBgRepeat == "" {
		userBlog.HeaderNavBgRepeat = "no-repeat"
	}
	if userBlog.PageBgRepeat == "" {
		userBlog.PageBgRepeat = "no-repeat"
	}
	userBlog.BlogNavPosition = this.normalizeBlogNavPosition(userBlog.BlogNavPosition)
	userBlog.HeaderNavBgImage = this.normalizeBlogBgImage(userBlog.HeaderNavBgImage)
	userBlog.PageBgImage = this.normalizeBlogBgImage(userBlog.PageBgImage)
}

func (this *BlogService) appendBgBlock(css *strings.Builder, selector, pseudoPosition, color, image string, opacity int, size, repeat string) {
	color = strings.TrimSpace(color)
	image = this.normalizeBlogBgImage(image)
	if color == "" && image == "" {
		return
	}

	opacityVal := this.normalizeBlogBgOpacity(opacity)
	sizeVal := this.normalizeBlogBgSize(size)
	repeatVal := this.normalizeBlogBgRepeat(repeat)

	css.WriteString(selector + " {\n")
	if color != "" {
		css.WriteString("  background-color: " + color + ";\n")
	}
	if image != "" {
		css.WriteString("  position: relative;\n")
	}
	css.WriteString("}\n")

	if image == "" {
		return
	}

	css.WriteString(selector + "::before {\n")
	css.WriteString("  content: '';\n")
	css.WriteString("  position: " + pseudoPosition + ";\n")
	css.WriteString("  top: 0; left: 0; right: 0; bottom: 0;\n")
	css.WriteString(fmt.Sprintf("  background-image: url('%s');\n", image))
	css.WriteString(fmt.Sprintf("  background-size: %s;\n", sizeVal))
	css.WriteString(fmt.Sprintf("  background-repeat: %s;\n", repeatVal))
	css.WriteString("  background-position: center center;\n")
	css.WriteString(fmt.Sprintf("  opacity: %.2f;\n", opacityVal))
	css.WriteString("  pointer-events: none;\n")
	if pseudoPosition == "fixed" {
		css.WriteString("  z-index: 0;\n")
	} else {
		css.WriteString("  z-index: 0;\n")
	}
	css.WriteString("}\n")

	if pseudoPosition != "fixed" {
		css.WriteString(selector + " > * {\n")
		css.WriteString("  position: relative;\n")
		css.WriteString("  z-index: 1;\n")
		css.WriteString("}\n")
	}
}

func (this *BlogService) BuildBlogStyleCss(userBlog info.UserBlog) string {
	this.fixUserBlogBackground(&userBlog)

	var css strings.Builder

	hasHeaderBg := strings.TrimSpace(userBlog.HeaderNavBgColor) != "" || userBlog.HeaderNavBgImage != ""
	if hasHeaderBg {
		css.WriteString("#headerContainer { background-color: transparent !important; }\n")
		css.WriteString("#headerAndNav .navbar-default { background: transparent !important; border-color: transparent; }\n")
	}
	this.appendBgBlock(&css, "#headerAndNav", "absolute",
		userBlog.HeaderNavBgColor, userBlog.HeaderNavBgImage,
		userBlog.HeaderNavBgOpacity, userBlog.HeaderNavBgSize, userBlog.HeaderNavBgRepeat)

	hasPageBg := strings.TrimSpace(userBlog.PageBgColor) != "" || userBlog.PageBgImage != ""
	if hasPageBg {
		this.appendBgBlock(&css, "html", "fixed",
			userBlog.PageBgColor, userBlog.PageBgImage,
			userBlog.PageBgOpacity, userBlog.PageBgSize, userBlog.PageBgRepeat)
		css.WriteString("body { background: transparent !important; }\n")
	}

	css.WriteString("#postsContainer { background: transparent !important; position: relative; z-index: 1; }\n")
	css.WriteString("#postsContainer .each-post { position: relative; z-index: 1; }\n")
	css.WriteString(this.buildBlogNavLayoutCss(userBlog.BlogNavPosition))
	return css.String()
}

func (this *BlogService) buildBlogNavLayoutCss(position string) string {
	position = this.normalizeBlogNavPosition(position)
	var css strings.Builder

	css.WriteString("#posts.post-page-layout {\n")
	css.WriteString("  display: flex !important;\n")
	css.WriteString("  gap: 24px;\n")
	css.WriteString("  align-items: flex-start;\n")
	css.WriteString("  width: auto !important;\n")
	css.WriteString("  max-width: 1200px;\n")
	css.WriteString("  margin-left: auto;\n")
	css.WriteString("  margin-right: auto;\n")
	css.WriteString("}\n")

	css.WriteString("#posts.post-page-layout .post-main {\n")
	css.WriteString("  flex: 1;\n")
	css.WriteString("  min-width: 0;\n")
	css.WriteString("  position: relative;\n")
	css.WriteString("  z-index: 1;\n")
	css.WriteString("}\n")

	css.WriteString("#blogNav.blog-nav-sidebar {\n")
	css.WriteString("  display: block !important;\n")
	css.WriteString("  position: sticky !important;\n")
	css.WriteString("  top: 20px;\n")
	css.WriteString("  left: auto !important;\n")
	css.WriteString("  right: auto !important;\n")
	css.WriteString("  align-self: flex-start;\n")
	css.WriteString("  width: clamp(220px, 24vw, 300px);\n")
	css.WriteString("  min-width: 220px;\n")
	css.WriteString("  max-width: 300px;\n")
	css.WriteString("  flex: 0 0 auto;\n")
	css.WriteString("  max-height: calc(100vh - 40px);\n")
	css.WriteString("  overflow: hidden;\n")
	css.WriteString("  z-index: 2;\n")
	css.WriteString("  padding: 8px;\n")
	css.WriteString("  border-radius: 4px;\n")
	css.WriteString("  border: 1px solid #ebeff2;\n")
	css.WriteString("  background-color: rgba(255, 255, 255, 0.95) !important;\n")
	css.WriteString("  opacity: 1 !important;\n")
	css.WriteString("}\n")

	css.WriteString("#blogNav.blog-nav-sidebar #blogNavContent {\n")
	css.WriteString("  display: block !important;\n")
	css.WriteString("  max-height: calc(100vh - 90px);\n")
	css.WriteString("  overflow-y: auto;\n")
	css.WriteString("  overflow-x: hidden;\n")
	css.WriteString("  word-wrap: break-word;\n")
	css.WriteString("  overflow-wrap: anywhere;\n")
	css.WriteString("}\n")
	css.WriteString("#blogNav.blog-nav-sidebar #blogNavContent a {\n")
	css.WriteString("  display: block;\n")
	css.WriteString("  line-height: 1.45;\n")
	css.WriteString("  white-space: normal;\n")
	css.WriteString("}\n")

	if position == "right" {
		css.WriteString("#blogNav.blog-nav-sidebar { order: 2; }\n")
		css.WriteString("#posts.post-page-layout .post-main { order: 1; }\n")
	}

	css.WriteString("@media (max-width: 768px) {\n")
	css.WriteString("  #posts.post-page-layout {\n")
	css.WriteString("    flex-direction: column;\n")
	css.WriteString("  }\n")
	css.WriteString("  #blogNav.blog-nav-sidebar {\n")
	css.WriteString("    position: relative !important;\n")
	css.WriteString("    top: auto !important;\n")
	css.WriteString("    width: 100%;\n")
	css.WriteString("    max-height: 220px;\n")
	css.WriteString("    order: -1;\n")
	css.WriteString("  }\n")
	css.WriteString("}\n")

	return css.String()
}

func (this *BlogService) UpdateUserBlogBackground(userId string, bg info.UserBlogBackground) bool {
	if bg.HeaderNavBgOpacity <= 0 {
		bg.HeaderNavBgOpacity = 100
	}
	if bg.PageBgOpacity <= 0 {
		bg.PageBgOpacity = 100
	}
	if bg.HeaderNavBgSize == "" {
		bg.HeaderNavBgSize = "cover"
	}
	if bg.PageBgSize == "" {
		bg.PageBgSize = "cover"
	}
	if bg.HeaderNavBgRepeat == "" {
		bg.HeaderNavBgRepeat = "no-repeat"
	}
	if bg.PageBgRepeat == "" {
		bg.PageBgRepeat = "no-repeat"
	}
	bg.BlogNavPosition = this.normalizeBlogNavPosition(bg.BlogNavPosition)
	return db.UpdateByQMap(db.UserBlogs, bson.M{"_id": bson.ObjectIdHex(userId)}, bg)
}
