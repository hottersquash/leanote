db.configs.updateMany({Key: "uploadImageSize"}, {$set: {ValueStr: "20"}});
db.configs.updateMany({Key: "uploadBlogLogoSize"}, {$set: {ValueStr: "5"}});
db.configs.updateMany({Key: "uploadBlogBgSize"}, {$set: {ValueStr: "20"}}, {upsert: true});
print("updated upload limits");
