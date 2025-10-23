# Contributing

Thanks for considering contributing to Sentinel. The project values clear, small PRs and good test coverage.

How to contribute

1. Fork the repository and create a topic branch for your work:

```bash
git checkout -b feat/my-feature
```

2. Keep commits focused and descriptive. Run `go fmt` before committing.

3. Add or update tests for your changes. The project uses Go's built-in testing framework.

4. Open a pull request targeting `main` with a clear description of what the change does and why.

Review and CI

- PRs should include tests where applicable.
- Maintain backwards compatibility where possible. If a breaking change is required, explain the migration steps in the PR description.

Code of conduct

Be respectful and collaborative. Follow any additional project maintainers' guidance during reviews.

Local development notes

- Copy `.env.example` to `.env` and fill the required configuration values (PORT, DATABASE_URL, JWT_SECRET) before running locally.
- Do not commit `.env`. Use GitHub Actions secrets (or other secret stores) for CI/deployment secrets.

