"""
ArmorClaw Default Skills

These skills are available by default in the ArmorClaw agent.
"""

from .ssl_tunnel_setup import (
    SSL_SKILLS,
    list_ssl_skills,
    setup_ssl_tunnel,
    get_ssl_status,
    NgrokTunnelSkill,
    CloudflareTunnelSkill,
    SelfSignedCertSkill
)

from .ssl_skill_handler import (
    SSL_SETUP_INSTRUCTIONS,
    DEFAULT_SKILLS
)

__all__ = [
    # SSL Skills
    'SSL_SKILLS',
    'list_ssl_skills',
    'setup_ssl_tunnel',
    'get_ssl_status',
    'NgrokTunnelSkill',
    'CloudflareTunnelSkill',
    'SelfSignedCertSkill',

    # Skill Handler
    'SSL_SETUP_INSTRUCTIONS',
    'DEFAULT_SKILLS',
]
