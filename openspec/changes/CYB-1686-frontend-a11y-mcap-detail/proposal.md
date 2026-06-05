# CYB-1686 Frontend A11y and MCAP Detail

## Summary

Fix audited frontend usability gaps:

- Add stable `id` / `name` attributes to AntD form controls that currently trigger accessibility warnings.
- Implement `/mcap-files/:id` as a real detail route instead of a title-only placeholder.

## Problem

The current UI has no JavaScript runtime error, but audits report form fields without identifiable names. In addition, `/mcap-files/:id` renders only a back button and page title, with no data request or detail content.

## Goals

- Representative pipeline, delivery, workflow, and MCAP form controls have stable `id` / `name` attributes.
- `/mcap-files/:id` reads the URL id, calls the existing MCAP API, and displays metadata plus related asset information.
- The MCAP detail route has loading, error, empty, and success states.

## Non-Goals

- Change backend MCAP APIs.
- Redesign the entire MCAP files list page.
- Add new form workflows beyond a11y attributes.
