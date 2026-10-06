python3 <<'PY'
from pathlib import Path

replacements = {
    "skillDomain.NormalizeManagedSkillFiles": "skillPackage.NormalizeManagedSkillFiles",
    "skillDomain.ManagedSkillStorageFiles": "skillPackage.ManagedSkillStorageFiles",
    "skillDomain.ManagedPackageAddressForSkill": "skillPackage.ManagedPackageAddressForSkill",
    "skillDomain.ManagedPackageLocatorForSkill": "skillPackage.ManagedPackageLocatorForSkill",
    "skillDomain.ManagedPackageAddressFromSkillLocator": "skillPackage.ManagedPackageAddressFromSkillLocator",
    "skillDomain.DecodeSkillDocument": "skillSource.DecodeSkillDocument",
    "skillDomain.ParseSkillDocument": "skillSource.ParseSkillDocument",
    "skillDomain.ManagedSkillDocument": "skillSource.ManagedSkillDocument",
    "skillDomain.SkillArtifactKind": "skillSource.SkillArtifactKind",
    "skillDomain.MarkdownDecoderID": "skillSource.MarkdownDecoderID",
    "skillDomain.SkillSchemaID": "skillSource.SkillSchemaID",
    "skillDomain.SkillSchemaVersion": "skillSource.SkillSchemaVersion",
    "skillDomain.BuiltInInstallerName": "skillSource.BuiltInInstallerName",
    "skillDomain.HydrationSchemaVersion": "skillSource.HydrationSchemaVersion",
    "skillDomain.IsSkillDefinitionFile": "skillSource.IsSkillDefinitionFile",
    "skillDomain.IsSkillKind": "skillSource.IsSkillKind",
    "skillDomain.SkillDefinitionFileName": "skillSource.SkillDefinitionFileName",
    "skillDomain.SkillDeclarationFromDefinition": "skillSource.SkillDeclarationFromDefinition",
    "skillDomain.SourceDocumentLocator": "skillSource.SourceDocumentLocator",
    "skillDomain.RuntimePackageLocator": "skillSource.RuntimePackageLocator",
}

for path in Path(".").rglob("*.go"):
    value = path.read_text()
    if "skillDomain." not in value:
        continue
    for old, new in replacements.items():
        value = value.replace(old, new)
    path.write_text(value)
PY
