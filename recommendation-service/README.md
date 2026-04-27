# recommendation-service

Python service that builds personalized workbooks from:

- user knowledge profile from `statistic-service`
- content catalog from `content-service`
- optional course-specific task calibration from `statistic-service`
- subject-specific tag values from MongoDB

It returns:

- weak tag vector
- selected theory blocks
- selected tasks
- ordered workbook items
- generated LaTeX workbook body
