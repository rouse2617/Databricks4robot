"""Semantic managers — each wraps one or more generated API classes.

Adapted from Stripe's per-resource service modules (e.g. _customer_service.py).
Each manager is stateless — holds only a reference to the shared APIRequestor.
"""
