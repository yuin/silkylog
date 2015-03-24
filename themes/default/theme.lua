config {
  extra_files = {
    {src = "css/*", dst = "statics/css/", template = false},
    {src = "js/*", dst = "statics/js/", template = false},
    {src = "img/*", dst = "statics/img/", template = false}
  }
}

about_this_site = [[
  <div>
  <p><i class="icon-male"></i> Author: Your Name</p>
  <p>Your profile</p>
  </div>
]]

find_me_on = [[
  <ul class="icons-ul">
    <li style="margin-bottom: 0.5em;"><i class="icon-li icon-large icon-github"></i><a href="https://github.com/example">GitHub</a></li>
  </ul>
]]

function seealso(art, tags)
  local buf = {"<ul><h3>See Also</h3>"}
  local seen = {}
  seen[art.permlink_path] = 1
  local i = 0
  for j, tag in ipairs(art.tags or {}) do
    for k, article in ipairs(tags[tag] or {}) do
      if seen[article.permlink_path] == nil then
        table.insert(buf, string.format("<li><a href=\"%s\">%s</a></li>", article.permlink_path, silkylog.htmlescape(article.title)))
        seen[article.permlink_path] = 1
        i = i + 1
        if i > 4 then
          break
        end
      end
    end
    if i > 4 then
      break
    end
  end
  table.insert(buf, "</ul>")
  if i == 0 then
    return ""
  end
  return table.concat(buf, "\n")
end
