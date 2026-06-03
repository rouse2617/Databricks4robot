# CYB-1622 component port layout

## Problem
Component port rows overflow in the component modal when input and output port editors are rendered side by side.

## Proposed Change
- Stack input and output port editors vertically in constrained modals.
- Make each port row responsive and keep remove actions inside the row.
- Keep output file path guidance as helper copy instead of a wide input placeholder.
