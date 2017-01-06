TODO
====
- option to group by source file
- option to ignore some package/type/function (using regexp)
- option to omit unexported funcs or methods
- option to ignore standard/vendored packages
- option to show package paths (not just names)
- use different style for methods and funcs
  + *for example:* darker/lighter background color
- connect anon funcs with their parents
  + *for example:* combine them together in one box representing parent func
- vary colors of each individual non-focused package
  + *for example:* use different hue for different package
- allow combination of grouping options
  + *for example:* group by pkg and type
- support multiple package paths for limit flag
- store call graph data and update only when source changes
  + *for example:* keep the data in /tmp folder or in working directory
