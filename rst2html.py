#vim: fileencoding=utf8
from __future__ import print_function
import sys, os, cgi, re
from os.path import abspath, dirname

from docutils.writers.html4css1 import HTMLTranslator, Writer
from docutils.core import publish_parts
from docutils import frontend, nodes, utils, writers, languages, io
from docutils.parsers.rst import directives, Directive

import sys
if sys.version_info[0] >= 3:
  string_types, integer_types = (str, bytes), (int,)
  py3 = True
else:
  py3 = False
  string_types, integer_types = (basestring, (int, long))
  bytes, str, range, input = (str, unicode, xrange, raw_input)
  from itertools import imap as map, ifilter as filter, izip as zip, ifilter as filter

# Directive for google code prittify
class CodeDirective(Directive):
  has_content = True
  option_spec = {'options': directives.unchanged}
  def run(self):
    self.assert_has_content()
    text = u"<pre class=\"prettyprint {}\">{}</pre>".format(self.options.get("options", u""), self._get_escaped_content())
    return [nodes.raw('', text, format='html')]
  def _get_escaped_content(self):
    return u'\n'.join(map(cgi.escape, self.content))

directives.register_directive("sourcecode", CodeDirective)

class HTML5Translator(HTMLTranslator):
  def visit_literal(self, node):
    classes = node.get('classes', [])
    if 'code' in classes:
      node['classes'] = [cls for cls in classes if cls != 'code']
      self.body.append(self.starttag(node, 'code', ''))
      return
    self.body.append(
      self.starttag(node, 'code', '', CLASS=''))
    text = node.astext()
    for token in self.words_and_spaces.findall(text):
      if token.strip():
        if self.sollbruchstelle.search(token):
          self.body.append(u'<span class="pre">%s</span>' % self.encode(token))
        else:
          self.body.append(self.encode(token))
      elif token in ('\n', ' '):
        self.body.append(token)
      else:
        self.body.append(u'&nbsp;' * (len(token) - 1) + ' ')
    self.body.append(u'</code>')
    raise nodes.SkipNode

  def visit_section(self, node):
    self.section_level += 1
    self.body.append(u"<section>\n")

  def depart_section(self, node):
    self.section_level -= 1
    self.body.append(u"</section>\n")

  def visit_title(self, node):
    check_id = 0
    close_tag = u'</p>\n'
    if isinstance(node.parent, nodes.topic):
      self.body.append(
          self.starttag(node, 'p', '', CLASS=''))
    elif isinstance(node.parent, nodes.sidebar):
      self.body.append(
          self.starttag(node, 'p', '', CLASS=''))
    elif isinstance(node.parent, nodes.Admonition):
      self.body.append(
          self.starttag(node, 'p', '', CLASS=''))
    elif isinstance(node.parent, nodes.table):
      self.body.append(
          self.starttag(node, 'caption', ''))
      close_tag = u'</caption>\n'
    elif isinstance(node.parent, nodes.document):
      self.body.append(self.starttag(node, 'h1', '', CLASS=''))
      close_tag = u'</h1>\n'
      self.in_document_title = len(self.body)
    else:
      assert isinstance(node.parent, nodes.section)
      h_level = self.section_level + self.initial_header_level - 1
      atts = {}
      if (len(node.parent) >= 2 and
        isinstance(node.parent[1], nodes.subtitle)):
        atts['CLASS'] = ''
      self.body.append(
          self.starttag(node, u'h%s' % h_level, '', **atts))
      atts = {}
      if node.hasattr('refid'):
        atts['class'] = ''
        atts['href'] = '#' + node['refid']
      if atts:
        close_tag = u'</h%s>\n' % (h_level)
      else:
        close_tag = u'</h%s>\n' % (h_level)
    self.context.append(close_tag)

  def write_colspecs(self):
    for node in self.colspecs:
      self.body.append(self.emptytag(node, 'col'))
    self.colspecs = []

  def visit_table(self, node):
    classes = ''
    self.body.append(
      self.starttag(node, 'table', CLASS=classes))

if __name__ == '__main__':
  if py3:
    text = sys.stdin.buffer.read().decode("utf8")
  else:
    text = sys.stdin.read().decode("utf8")

  writer = Writer()
  writer.translator_class = HTML5Translator
  data = publish_parts(source=text, writer=writer)
  html = re.sub(u"<div[^>]+>(.*)", u"\\1", data["html_body"], 1)[:-7].strip()
  if py3:
    sys.stdout.buffer.write(html.encode("utf8"))
  else:
    print(html.encode("utf8"))
