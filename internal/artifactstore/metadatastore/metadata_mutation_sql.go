package metadatastore

const updateRootSQL = `
	UPDATE artifact_roots
	   SET display_name = ?,
	       description = ?,
	       enabled = ?,
	       data_schema_id = ?,
	       data_json = ?,
	       modified_at = ?,
	       soft_deleted_at = ?
	 WHERE root_id = ?
	   AND modified_at = ?`

const updateSourceSQL = `
	UPDATE artifact_sources
	   SET display_name = ?,
	       enabled = ?,
	       config_schema_id = ?,
	       config_json = ?,
	       last_observed_generation = ?,
	       last_scanned_at = ?,
	       observation_revision = ?,
	       diagnostics_json = ?,
	       modified_at = ?
	 WHERE source_id = ?
	   AND modified_at = ?`

const deleteSourceSQL = `
	DELETE FROM artifact_sources
	 WHERE source_id = ?
	   AND modified_at = ?`

const updateRootSourceAttachmentSQL = `
	UPDATE root_source_attachments
	   SET role = ?,
	       priority = ?,
	       enabled = ?,
	       data_schema_id = ?,
	       data_json = ?,
	       modified_at = ?
	 WHERE root_id = ?
	   AND source_id = ?
	   AND modified_at = ?`

const deleteRootSourceAttachmentSQL = `
	DELETE FROM root_source_attachments
	 WHERE root_id = ?
	   AND source_id = ?
	   AND modified_at = ?`

const updateCollectionSQL = `
	UPDATE artifact_collections
	   SET display_name = ?,
	       description = ?,
	       enabled = ?,
	       data_schema_id = ?,
	       data_json = ?,
	       modified_at = ?,
	       soft_deleted_at = ?
	 WHERE collection_id = ?
	   AND modified_at = ?`

const updateRecordSQL = `
	UPDATE artifact_records
	   SET collection_id = ?,
	       record_mode = ?,
	       tracking_mode = ?,
	       pinned_definition_digest = ?,
	       last_resolved_definition_digest = ?,
	       enabled = ?,
	       data_schema_id = ?,
	       data_json = ?,
	       state = ?,
	       diagnostics_json = ?,
	       modified_at = ?
	 WHERE record_id = ?
	   AND modified_at = ?`

const deleteRecordSQL = `
	DELETE FROM artifact_records
	 WHERE record_id = ?
	   AND modified_at = ?`

const publishSourceObservationSQL = `
	UPDATE artifact_sources
	   SET last_observed_generation = ?,
	       last_scanned_at = ?,
	       diagnostics_json = ?
	 WHERE source_id = ?
	   AND modified_at = ?`
