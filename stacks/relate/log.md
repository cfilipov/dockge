# RELATE

- **Source**: https://github.com/inducer/relate
- **Image**: python:3.11-slim (generic Python runtime)
- **Description**: Web-based courseware platform by Andreas Kloeckner. Django app for course flow management with Git-based content versioning.
- **Services**: relate (Django app), postgres
- **Notes**: No official Docker image. Uses generic Python image. Course content stored in Git repositories (git-roots volume). RELATE supports homework, quizzes, grading workflows.
