当包含图片输入时 (when image input exists)：
- 先提取可见文字 (extract visible text first)
- 再转换成模板草稿结构 (rewrite into template draft format)
- OCR 不确定性、截断风险写入 `warnings`
