# pinazu-core

## Description
Core Service for orchestrating and managing AI Generative Application workflows.

## Architecture

![Conceptual Architecture](/docs/img/conceptual_architecture.png)

Conceptual of the Core Architecture interact in an microservices environemnts.

![Core Architecture](/docs/img/coreapp_architecture.png)

Core Architecture represent the components and their interactions.

## Badges
On some READMEs, you may see small images that convey metadata, such as whether or not all the tests are passing for the project. You can use Shields to add some to your README. Many services also have instructions for adding a badge.

## Visuals
Depending on what you are making, it can be a good idea to include screenshots or even a video (you'll frequently see GIFs rather than actual videos). Tools like ttygif can help, but check out Asciinema for a more sophisticated method.

## Installation
Within a particular ecosystem, there may be a common way of installing things, such as using Yarn, NuGet, or Homebrew. However, consider the possibility that whoever is reading your README is a novice and would like more guidance. Listing specific steps helps remove ambiguity and gets people to using your project as quickly as possible. If it only runs in a specific context like a particular programming language version or operating system or has dependencies that have to be installed manually, also add a Requirements subsection.

## Usage

1. On one terminal run the following command to build the project and setup the development environment:
```bash
make build

# To run the development environment
make run
```

2. Open another terminal and run the following command to run the core application:
```bash
pinazu serve all -c configs/config.yaml
```
## Integration Test

1. Install Node.js dependencies (for integration testing)

cd e2e && npm install && cd ../..

2. Run integration tests (requires services to be running)
cd e2e && npm run test:integration

3. Integration tests (requires services running)

cd e2e && npm run test:integration            # Run all Playwright integration tests
cd e2e && npm run test:flows                  # Run flows API tests only
cd e2e && npm run test:integration:ui         # Run tests with Playwright UI
./scripts/run-api-tests.sh          # Run integration tests with automatic setup# Install Node.js dependencies (for integration testing)
cd e2e && npm install && cd ../..


4. Run integration tests (requires services to be running)

cd e2e && npm run test:integration

5.  Integration tests (requires services running)

cd e2e && npm run test:integration            # Run all Playwright integration tests
cd e2e && npm run test:flows                  # Run flows API tests only
cd e2e && npm run test:integration:ui         # Run tests with Playwright UI
./scripts/run-api-tests.sh          # Run integration tests with automatic setup


**WARNING**
You can modify the configuration file `config.yaml` inside the `config` folder to change the configuration of the application.
After that, you can run this command to apply the custom configs:
```bash
pinazu serve all -c configs/config.yaml
```

## Support
Tell people where they can go to for help. It can be any combination of an issue tracker, a chat room, an email address, etc.

## Roadmap
If you have ideas for releases in the future, it is a good idea to list them in the README.

## Contributing
State if you are open to contributions and what your requirements are for accepting them.

For people who want to make changes to your project, it's helpful to have some documentation on how to get started. Perhaps there is a script that they should run or some environment variables that they need to set. Make these steps explicit. These instructions could also be useful to your future self.

You can also document commands to lint the code or run tests. These steps help to ensure high code quality and reduce the likelihood that the changes inadvertently break something. Having instructions for running tests is especially helpful if it requires external setup, such as starting a Selenium server for testing in a browser.

## Authors and acknowledgment
Show your appreciation to those who have contributed to the project.

## License
For open source projects, say how it is licensed.

## Project status
If you have run out of energy or time for your project, put a note at the top of the README saying that development has slowed down or stopped completely. Someone may choose to fork your project or volunteer to step in as a maintainer or owner, allowing your project to keep going. You can also make an explicit request for maintainers.
