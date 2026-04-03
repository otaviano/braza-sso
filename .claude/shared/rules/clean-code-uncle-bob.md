---
name: clean-code-uncle-bob
description: Clean Code principles by Uncle Bob (Robert C. Martin). Applies to all source files to enforce readability, small functions, meaningful names, and low WTF-per-minute density.
applyTo: "**/*.{cs,go,ts,tsx,js,jsx,py}"
---

# Clean Code (Uncle Bob)
Description: Writing code that is elegant, efficient, easy to read, and easy to maintain. Clean code is focused on being understandable by humans, not just machines. It is written by developers who care about craftsmanship and want to reduce "WTFs per minute".

Core Principles:
- Boy Scout Rule: "Always leave the campground cleaner than you found it." Improve code quality continuously.
- Single Responsibility Principle (SRP): A class or method should have one, and only one, reason to change.
- DRY (Don't Repeat Yourself): Avoid duplication of logic.
- Small Functions: Functions should be small, and then smaller than that. Do one thing, do it well, and do it only.
- Meaningful Names: Names should reveal intent. Avoid cryptic abbreviations and magic numbers/strings.
- Expressive Code: Code should read like well-written prose.

Best Practices:
- Keep functions short (ideally < 20 lines).
- Minimize the number of arguments (zero or one is ideal, avoid three or more).
- No side effects: A function should not do something hidden.
- Avoid comments that explain what the code does; use good naming instead.
- Use comments only to explain why or to warn about potential issues.
- Clean code is self-documenting.
- Use exceptions rather than return codes.
- Don't return null; throw an exception or return a special case object.
- TDD (Test Driven Development) is highly recommended.
- Keep tests clean; tests are as important as production code.
- Tests ensure code is flexible, maintainable, and reusable.

Code Structure (Boy Scout Rule Application):
- Vertical Formatting: Similar to newspaper articles, with high-level concepts at the top and low-level details below.
- Refactoring: Constantly improve the design of code without changing behavior.

Anti-Patterns to Avoid:
- Magic Numbers: Unnamed hard-coded numbers.
- Comments as "Makeup": Using comments to cover bad code.
- High "WTF" Density: Code that makes other developers ask "what the f***" when reading it.

References:
- Clean Code: A Handbook of Agile Software Craftsmanship by Robert C. Martin.
- The Clean Coder: A Code of Conduct for Professional Programmers by Robert C. Martin.
